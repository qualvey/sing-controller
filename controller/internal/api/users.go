package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/sagernet/sing-box/option"
	"github.com/qualvey/sing-controller/internal/store"
	"github.com/sagernet/sing/common/json"
)

// 支持 users[] 的入站类型 → 用户字段映射
var userSupportedInboundTypes = map[string][]string{
	"vless":      {"name", "uuid", "flow"},
	"vmess":      {"name", "uuid"},
	"trojan":     {"name", "password"},
	"tuic":       {"name", "uuid", "password"},
	"hysteria2":  {"name", "password"},
	"hysteria":   {"name", "password"},
	"shadowsocks": {"name", "password"},
	"anytls":     {"name", "password"},
	"shadowtls":  {"name", "password"},
}

// buildUserForType 按入站类型构建用户 JSON（只含该类型支持的字段）
func buildUserForType(user store.UserMeta, inboundType string) map[string]any {
	fields, ok := userSupportedInboundTypes[inboundType]
	if !ok {
		return nil
	}
	u := map[string]any{"name": user.Name}
	for _, f := range fields {
		switch f {
		case "uuid":
			if user.UUID != "" {
				u["uuid"] = user.UUID
			}
		case "password":
			if user.Password != "" {
				u["password"] = user.Password
			}
		case "flow":
			if user.Flow != "" {
				u["flow"] = user.Flow
			}
		}
	}
	return u
}

// syncUsersToInbounds 用用户池重建所有绑定入站的 users[]（用户池为唯一真相）
func syncUsersToInbounds(ctx context.Context, options *option.Options, users []store.UserMeta) error {
	for i := range options.Inbounds {
		inbound := &options.Inbounds[i]
		if _, ok := userSupportedInboundTypes[inbound.Type]; !ok {
			continue
		}
		// 绑定到该入站的用户
		var bound []map[string]any
		for _, u := range users {
			for _, tag := range u.BoundInbounds {
				if tag == inbound.Tag {
					if item := buildUserForType(u, inbound.Type); item != nil {
						bound = append(bound, item)
					}
					break
				}
			}
		}
		if err := setInboundUsers(ctx, inbound, bound); err != nil {
			return err
		}
	}
	return nil
}

// setInboundUsers 通过 JSON 层重写 inbound 的 users[]（保留其他字段）
func setInboundUsers(ctx context.Context, inbound *option.Inbound, users []map[string]any) error {
	content, err := json.Marshal(inbound)
	if err != nil {
		return err
	}
	var decoded map[string]any
	if err := json.Unmarshal(content, &decoded); err != nil {
		return err
	}
	if len(users) == 0 {
		delete(decoded, "users")
	} else {
		decoded["users"] = users
	}
	content, err = json.Marshal(decoded)
	if err != nil {
		return err
	}
	var fresh option.Inbound
	if err := json.UnmarshalContext(ctx, content, &fresh); err != nil {
		return err
	}
	*inbound = fresh
	return nil
}

func (h *Handler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"users": h.store.Users()})
}

// userBody 用户创建/更新请求体
type userBody struct {
	Name          string   `json:"name"`
	UUID          string   `json:"uuid,omitempty"`
	Password      string   `json:"password,omitempty"`
	Flow          string   `json:"flow,omitempty"`
	BoundInbounds []string `json:"bound_inbounds,omitempty"`
}

func (b *userBody) validate() error {
	if b.Name == "" {
		return errors.New("用户 name 必填")
	}
	if b.UUID == "" && b.Password == "" {
		return errors.New("uuid 和 password 至少填一个")
	}
	return nil
}

func (h *Handler) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var body userBody
	raw, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := json.UnmarshalContext(h.ctx(r), raw, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := body.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		for _, u := range meta.Users {
			if u.Name == body.Name {
				return errors.New("用户已存在: " + body.Name)
			}
		}
		meta.Users = append(meta.Users, store.UserMeta{
			Name:          body.Name,
			UUID:          body.UUID,
			Password:      body.Password,
			Flow:          body.Flow,
			BoundInbounds: body.BoundInbounds,
		})
		return syncUsersToInbounds(h.ctx(r), options, meta.Users)
	})
	writeJSON(w, http.StatusOK, map[string]any{"saved": true})
}

func (h *Handler) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var body userBody
	raw, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := json.UnmarshalContext(h.ctx(r), raw, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if body.Name != "" && body.Name != name {
		writeError(w, http.StatusBadRequest, errors.New("不允许重命名用户"))
		return
	}
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		idx := -1
		for i, u := range meta.Users {
			if u.Name == name {
				idx = i
				break
			}
		}
		if idx < 0 {
			return errors.New("用户不存在: " + name)
		}
		updated := meta.Users[idx]
		updated.UUID = body.UUID
		updated.Password = body.Password
		updated.Flow = body.Flow
		updated.BoundInbounds = body.BoundInbounds
		meta.Users[idx] = updated
		return syncUsersToInbounds(h.ctx(r), options, meta.Users)
	})
	writeJSON(w, http.StatusOK, map[string]any{"saved": true})
}

func (h *Handler) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		found := false
		filtered := meta.Users[:0]
		for _, u := range meta.Users {
			if u.Name == name {
				found = true
				continue
			}
			filtered = append(filtered, u)
		}
		if !found {
			return errors.New("用户不存在: " + name)
		}
		meta.Users = filtered
		return syncUsersToInbounds(h.ctx(r), options, meta.Users)
	})
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

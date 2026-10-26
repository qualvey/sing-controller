#!/usr/bin/env bash
# local-cd.sh — 本地构建 linux-systemd deb 并部署到远端
#
# 用法：
#   ./local-cd.sh              # 构建 amd64、arm64、armv7，并部署匹配远端的包
#   ./local-cd.sh arm          # 只构建 arm64、armv7，并部署到 ARM 远端
#   ./local-cd.sh --skip-frontend
#   ./local-cd.sh arm --skip-frontend
#
# 首次运行会询问 SSH 信息，并保存到项目根目录的 .deploy.env（已加入 .gitignore）。

set -Eeuo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_CONFIG="$ROOT/.deploy.env"
MODE="all"
SKIP_FRONTEND=0

log() {
    printf '\n\033[1;32m[*]\033[0m %s\n' "$*"
}

warn() {
    printf '\033[1;33m[!]\033[0m %s\n' "$*" >&2
}

die() {
    printf '\033[1;31m[错误]\033[0m %s\n' "$*" >&2
    exit 1
}

usage() {
    sed -n '2,10p' "$0"
}

for arg in "$@"; do
    case "$arg" in
        arm)
            MODE="arm"
            ;;
        --skip-frontend)
            SKIP_FRONTEND=1
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            usage >&2
            die "未知参数: $arg"
            ;;
    esac
done

require_command() {
    command -v "$1" >/dev/null 2>&1 || die "缺少命令 $1，请先安装后重试"
}

require_command go
require_command ssh
require_command scp
if [[ "$SKIP_FRONTEND" -eq 0 ]]; then
    require_command pnpm
fi

save_ssh_config() {
    local key_path

    printf '首次运行需要配置远端 SSH（密码不会保存，使用 SSH agent 或交互式输入）。\n'
    while :; do
        read -r -p 'SSH 主机/IP: ' SSH_HOST
        [[ -n "$SSH_HOST" ]] && break
        warn 'SSH 主机不能为空'
    done

    read -r -p 'SSH 用户 [root]: ' SSH_USER
    SSH_USER="${SSH_USER:-root}"

    read -r -p 'SSH 端口 [22]: ' SSH_PORT
    SSH_PORT="${SSH_PORT:-22}"
    [[ "$SSH_PORT" =~ ^[0-9]+$ ]] || die 'SSH 端口必须是数字'

    read -r -p 'SSH 私钥路径 [留空使用默认配置/agent]: ' key_path
    if [[ -n "$key_path" ]]; then
        SSH_KEY="${key_path/#\~/$HOME}"
        [[ -f "$SSH_KEY" ]] || die "私钥不存在: $SSH_KEY"
    else
        SSH_KEY=""
    fi

    umask 077
    {
        printf '# local-cd.sh SSH 配置；不要提交到 Git\n'
        printf 'SSH_HOST=%q\n' "$SSH_HOST"
        printf 'SSH_USER=%q\n' "$SSH_USER"
        printf 'SSH_PORT=%q\n' "$SSH_PORT"
        printf 'SSH_KEY=%q\n' "$SSH_KEY"
    } > "$DEPLOY_CONFIG"
    chmod 600 "$DEPLOY_CONFIG"
    log "SSH 配置已保存: $DEPLOY_CONFIG"
}

if [[ -f "$DEPLOY_CONFIG" ]]; then
    # shellcheck disable=SC1090
    source "$DEPLOY_CONFIG"
else
    save_ssh_config
fi

SSH_HOST="${SSH_HOST:-}"
SSH_USER="${SSH_USER:-}"
SSH_PORT="${SSH_PORT:-22}"
SSH_KEY="${SSH_KEY:-}"
[[ -n "$SSH_HOST" ]] || die "$DEPLOY_CONFIG 缺少 SSH_HOST，请删除后重新配置"
[[ -n "$SSH_USER" ]] || die "$DEPLOY_CONFIG 缺少 SSH_USER，请删除后重新配置"
[[ "$SSH_PORT" =~ ^[0-9]+$ ]] || die "$DEPLOY_CONFIG 中的 SSH_PORT 无效"
if [[ -n "$SSH_KEY" ]]; then
    [[ -f "$SSH_KEY" ]] || die "SSH 私钥不存在: $SSH_KEY"
fi

REMOTE="$SSH_USER@$SSH_HOST"
SSH_OPTS=(-p "$SSH_PORT" -o ConnectTimeout=10)
SCP_OPTS=(-P "$SSH_PORT")
if [[ -n "$SSH_KEY" ]]; then
    SSH_OPTS+=(-i "$SSH_KEY")
    SCP_OPTS+=(-i "$SSH_KEY")
fi

log "检查远端 SSH、dpkg 和 systemd: $REMOTE"
REMOTE_ARCH="$(ssh "${SSH_OPTS[@]}" "$REMOTE" 'uname -m')" || die 'SSH 连接失败'
REMOTE_ARCH="${REMOTE_ARCH//$'\r'/}"
REMOTE_ARCH="${REMOTE_ARCH//$'\n'/}"
ssh "${SSH_OPTS[@]}" "$REMOTE" \
    'command -v dpkg >/dev/null && command -v systemctl >/dev/null' \
    || die '远端必须安装 dpkg，并使用 systemd'

case "$REMOTE_ARCH" in
    x86_64|amd64)
        PACKAGE_ARCH="amd64"
        ;;
    aarch64|arm64)
        PACKAGE_ARCH="arm64"
        ;;
    armv7l|armv7|armhf)
        PACKAGE_ARCH="armv7"
        ;;
    *)
        die "不支持的远端 CPU 架构: $REMOTE_ARCH（支持 x86_64、aarch64、armv7l）"
        ;;
esac

if [[ "$MODE" == arm && "$PACKAGE_ARCH" == amd64 ]]; then
    die "已指定 arm，但远端是 x86_64；请去掉 arm 参数或更换远端主机"
fi

GORELEASER="$(command -v goreleaser || true)"
if [[ -z "$GORELEASER" ]]; then
    log '未找到 goreleaser，安装到 Go bin 目录'
    GOBIN="$(go env GOPATH)/bin"
    mkdir -p "$GOBIN"
    GOBIN="$GOBIN" go install github.com/goreleaser/goreleaser/v2@latest
    GORELEASER="$GOBIN/goreleaser"
fi

if [[ "$SKIP_FRONTEND" -eq 1 ]]; then
    log '跳过前端构建 (--skip-frontend)'
else
    log '安装前端依赖并构建 WebUI'
    (
        cd "$ROOT/controller/web"
        pnpm install --frozen-lockfile
        FONT="${FONT:-misans}" pnpm run build
    )
fi

BUILD_CONFIG="$ROOT/.goreleaser.yaml"
TEMP_CONFIG=""
cleanup() {
    if [[ -n "$TEMP_CONFIG" && -f "$TEMP_CONFIG" ]]; then
        rm -f "$TEMP_CONFIG"
    fi
}
trap cleanup EXIT

if [[ "$MODE" == arm ]]; then
    TEMP_CONFIG="$(mktemp "${TMPDIR:-/tmp}/sing-controller-goreleaser.XXXXXX.yaml")"
    # 从正式配置中只保留 arm64 和 armv7；其余 systemd/deb 打包配置不变。
    awk '
        /^    goarch:$/ {
            print
            print "      - arm64"
            print "      - arm"
            in_goarch = 1
            next
        }
        in_goarch && /^    [[:alpha:]_]/ { in_goarch = 0 }
        in_goarch { next }
        { print }
    ' "$ROOT/.goreleaser.yaml" > "$TEMP_CONFIG"
    BUILD_CONFIG="$TEMP_CONFIG"
fi

log "构建 linux-systemd deb（模式: $MODE；远端架构: $REMOTE_ARCH）"
(
    cd "$ROOT"
    "$GORELEASER" release --snapshot --skip=publish --clean --config "$BUILD_CONFIG"
)

shopt -s nullglob
DEB_PACKAGES=("$ROOT"/dist/*.deb)
shopt -u nullglob
(( ${#DEB_PACKAGES[@]} > 0 )) || die 'dist 下没有生成 .deb 安装包'

select_package() {
    local package deb_arch name
    for package in "${DEB_PACKAGES[@]}"; do
        name="$(basename "$package")"
        if command -v dpkg-deb >/dev/null 2>&1; then
            deb_arch="$(dpkg-deb -f "$package" Architecture 2>/dev/null || true)"
            case "$PACKAGE_ARCH:$deb_arch" in
                amd64:amd64|arm64:arm64|armv7:armhf)
                    printf '%s\n' "$package"
                    return 0
                    ;;
            esac
        fi
        case "$PACKAGE_ARCH:$name" in
            amd64:*_linux_amd64.deb)
                printf '%s\n' "$package"
                return 0
                ;;
            arm64:*_linux_arm64.deb)
                printf '%s\n' "$package"
                return 0
                ;;
            armv7:*_linux_armv7.deb|armv7:*_linux_arm_7.deb)
                printf '%s\n' "$package"
                return 0
                ;;
        esac
    done
    return 1
}

DEB="$(select_package || true)"
[[ -n "$DEB" ]] || die "没有找到与远端 $REMOTE_ARCH 匹配的 .deb（构建模式: $MODE）"
DEB_NAME="$(basename "$DEB")"
[[ "$DEB_NAME" =~ ^[A-Za-z0-9_.+-]+\.deb$ ]] || die "非法安装包文件名: $DEB_NAME"

log "部署 $DEB_NAME -> $REMOTE"
scp "${SCP_OPTS[@]}" "$DEB" "$REMOTE:/tmp/$DEB_NAME"

# config.json 虽然在 nfpm 中标记为 noreplace，但这里额外备份并恢复，保证
# 只要目标机已有配置文件，dpkg 升级过程就绝不会覆盖它，也不会交互询问。
# postinstall 会启用服务；这里再次 restart，确保升级后立即加载新版本。
ssh -tt "${SSH_OPTS[@]}" "$REMOTE" \
    "set -e
CONFIG_FILE=/etc/sing-controller/config.json
CONFIG_BACKUP=/tmp/sing-controller-config.json.\$\$
HAS_CONFIG=0
if sudo test -e \"\$CONFIG_FILE\"; then
    sudo cp -a \"\$CONFIG_FILE\" \"\$CONFIG_BACKUP\"
    HAS_CONFIG=1
fi
cleanup() {
    sudo rm -f /tmp/$DEB_NAME \"\$CONFIG_BACKUP\"
}
trap cleanup EXIT
sudo env DEBIAN_FRONTEND=noninteractive dpkg --force-confold -i /tmp/$DEB_NAME
if [ \"\$HAS_CONFIG\" = 1 ]; then
    sudo cp -a \"\$CONFIG_BACKUP\" \"\$CONFIG_FILE\"
fi
sudo systemctl restart sing-controller.service
sudo systemctl --no-pager --full status sing-controller.service"

log "部署完成: $REMOTE（$PACKAGE_ARCH）"
printf '查看日志: ssh -p %s %s@%s journalctl -u sing-controller -f\n' "$SSH_PORT" "$SSH_USER" "$SSH_HOST"

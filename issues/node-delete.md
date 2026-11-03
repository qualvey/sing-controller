# 节点删除时的边界条件

如果一个节点删除时，
同时被至少一个selector引用，这个节点删除之后，该selector的outbounds就是空，此时sing-box是无法启动的。



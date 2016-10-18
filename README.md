# lrconf

Light Remote Configuracion system

`lrconf` is a lightweight remote configuration management tool  has 2 elements.

lrconf-agent: The lightest configuration agent on:
lrconf-server: a config file server with all config files already formatted in its final format

These 2 elemens are focused on

* keeping local configuration files up-to-date using data stored in the Lrconf-server,
* reloading applications to pick up new config file changes

#Install lrconf-agent

You can just create a directory to place binary on /opt/lrconf-agent/  and copy (and reconfigure if neede) a minimal config from pkg/agent/conf/lrconf-agent-sample_min.toml to /opt/lrconf-agent/lrconf-agent.toml. The agent itself will download the full lrconf-agent config from the server ( use pkg/agent/conf/lrconf-agent-sample_node.toml as the full configuration config in the server side as described below)


#NOTE on lrconf-server

Not yet provided  in this packet instead now you can test with apache server (with php) and make yourself the directory tree.
```
$DOCUMENT_ROOT/upload/index.php (provided in the tools dir)
$DOCUMENT_ROOT/nodes/<node_id>/lronf-agent.toml  (customized lrconf-agent conf for only this node)
$DOCUMENT_ROOT/nodes/<node_id>/<checkid>/file1.conf
$DOCUMENT_ROOT/nodes/<node_id>/<checkid>/file2.conf
...
...
$DOCUMENT_ROOT/nodes/<node_id>/<checkid>/fileN.conf
```




## Run from master
If you want to build a package yourself, or contribute. Here is a guide for how to do that.

### Dependencies

- Go 1.5

### Get Code

```
go get github.com/toni-moreno/lrconf
```

### Building
```
cd $GOPATH/src/github.com/toni-moreno/lrconf
make
```

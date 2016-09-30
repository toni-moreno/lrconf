# lrconf

Light Remote Configuracion system

`lrconf` is a lightweight remote configuration management tool  has 2 elements.

lrconf-agent: The lightest configuration agent on:
lrconf-server: a config file server with all information needed

These 2 elemens are focused on

* keeping local configuration files up-to-date using data stored in the Lrconf-server,
* reloading applications to pick up new config file changes



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

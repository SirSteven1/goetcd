# goetcd
### 简单服务发现Golang代码操作
#### etcd 安装

```
   # 官网
   https://github.com/etcd-io/etcd/tree/main/client/v3
   https://pkg.go.dev/github.com/coreos/etcd/clientv3
   
   # 安装依赖
   go get go.etcd.io/etcd/client/v3
   
   # 安装etcd
   [root@node01 ~]# yum install -y etcd
   
   # 设置开机自动启动
   systemctl enable etcd
   
   # 启动etcd
   systemctl start etcd

   # 查看etcd运行状态
   systemctl status etcd

# systemd配置
从systemctl status etcd命令的输出可以看到，etcd的 systemd配置文件位于/usr/lib/systemd/system/etcd.service，该配置文件的内容如下：

$ cat /usr/lib/systemd/system/etcd.service
[Unit]
Description=Etcd Server
After=network.target
After=network-online.target
Wants=network-online.target

[Service]
Type=notify
WorkingDirectory=/var/lib/etcd/
EnvironmentFile=-/etc/etcd/etcd.conf
User=etcd
# set GOMAXPROCS to number of processors
ExecStart=/bin/bash -c "GOMAXPROCS=$(nproc) /usr/bin/etcd --name=\"${ETCD_NAME}\" --data-dir=\"${ETCD_DATA_DIR}\" --listen-client-urls=\"${ETCD_LISTEN_CLIENT_URLS}\""
Restart=on-failure
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target

# 从上面的配置中可以看到，etcd的配置文件位于/etc/etcd/etcd.conf，如果我们想要修改某些配置项，可以编辑该文件。

# 远程访问
etcd安装完成后，默认只能本地访问，如果需要开启远程访问，还需要修改/etc/etcd/etcd.conf中的配置。例如，本实例中我安装etcd的机器IP是10.103.18.41，我尝试通过自己的机器远程访问10.103.18.41上安装的etcd的2379端口，结果访问被拒绝：

# 修改/etc/etcd/etcd.conf配置：
ETCD_LISTEN_CLIENT_URLS="http://10.103.18.41:2379,http://localhost:2379"

# 然后重启
systemctl restart etcd
```
#### 连接etcd

```
    package main
    
    import(
       "context"
       "fmt"
       "time"
       clientv3 "go.etcd.io/etcd/client/v3"
    )
    
    var (
        config clientv3.Config
        client *clientv3.Client
        err error
        kv clientv3.KV
        putResp *clientv3.PutResponse
        getResp *clientv3.GetResponse
        delResp *clientv3.DeleteResponse
        leaseID  clientv3.LeaseID
        LeaseGrantResp *clientv3.LeaseGrantResponse
        keepResp *clientv3.LeaseKeepAliveResponse
        keepRespChan <-chan *clientv3.LeaseKeepAliveResponse //只读管道
    )
    
    func main(){
        //ETCD客户端连接信息
        config = clientv3.Config{
            Endponts: []string{"127.0.0.1:2379"}, //节点信息
            DialTimeout:5*time.Second, //超时时间
        }
        
        //建立连接
        if client,err = clientv3.New(config);err!=nil{
            fmt.Println(err)
            return
        }
        fmt.Println(client)
        
        //用于读写ETCD的键值
        kv = clientv3.NewKV(client)
        
        // 操作etcd,context.TODO() 这是一个上下文,如果这上下文不知道选那种,就选这个万精油;clientv3.WithPrevKV()加这参数获取前一个kv的值
        if putResp,err = kv.Put(context.TODO(),"/demo/example/etcd","123",clientv3.WithPrevKV());err!=nil{
           fmt.Println(err)
           return
        }
        
        // Revision: 作用域为集群，逻辑时间戳，全局单调递增，任何 key 修改都会使其自增
        fmt.Println("Revision is:", putResp.Header.Revision)
        
        if putResp.PreKv!=nil{
           //查看被更新的kv
           fmt.Println("更新的Key是：", string(putResp.PrevKv.Key))
		       fmt.Println("被更新的Value是：", string(putResp.PrevKv.Value))
        }
        
        //读取ETCD数据
        if getResp, err = kv.Get(context.TODO(), "/demo/example/etcd"); err != nil {
          fmt.Println(err)
          return
        }
        fmt.Println(getResp.Kvs)

        // 读取ETCD数据，获取前缀相同的WithPrefix()
        if getResp, err = kv.Get(context.TODO(), "/demo/example/", clientv3.WithPrefix()); err != nil {
          fmt.Println(err)
          return
        }
        fmt.Println(getResp.Kvs)

        // 删除ETCD数据;WithPrevKV--->赋值数据给delResp.PrevKvs,方便后续判断
        // 删除多个key：kv.Delete(context.TODO(), "/demo/example/", clientv3.WithPrefix())
        if delResp, err = kv.Delete(context.TODO(), "/demo/example/etcd", clientv3.WithPrevKV()); err != nil {
          fmt.Println(err)
          return
        }
        
        // 打印被删除之前的kv
        if len(delResp.PrevKvs) != 0 {
          for _, kvpx := range delResp.PrevKvs {
            fmt.Println("被删除的数据是: ", string(kvpx.Key), string(kvpx.Value))
          }
        }
        
        //租约、自动租约、lease
        //申请租约
        lease:=clientv3.Lease(client)
        
        // 申请一个10s的租约
        if LeaseGrantResp, err = lease.Grant(context.TODO(), 10); err != nil {
          fmt.Println("租约申请失败", err)
          return
        }

        // 租约ID
        leaseID = LeaseGrantResp.ID

        // 自动续租
        if keepRespChan, err = lease.KeepAlive(context.TODO(), leaseID); err != nil {
          fmt.Println("自动续租失败", err)
          return
        }
        
        /* 
            10s后自动过期
            ctx, canceFunc := context.WithCancel(context.TODO())
            // 自动续租
            if keepRespChan, err = lease.KeepAlive(ctx, leaseID); err != nil {
                fmt.Println("自动续租失败", err)
              return
            }
            canceFunc()

          */
          
          // 处理续约应答的协程  消费keepRespChan
          go func() {
            for {
              select {
              case keepResp = <-keepRespChan:
                if keepRespChan == nil {
                  fmt.Println("租约已经失效了")
                  goto END
                } else {
                  // KeepAlive每秒会续租一次,所以就会收到一次应答
                  fmt.Println("收到应答,租约ID是:", keepResp.ID)
                }
              }
            }
          END:
          }()

          // put一个kv,让他与租约关联起来,从而实现10s后自动过期,key就会被删除; 关联用的是clientv3.WithLease(leaseID)
          if putResp, err = kv.Put(context.TODO(), "/cron/lock/job3", "3", clientv3.WithLease(leaseID)); err != nil {
            fmt.Println(err)
            return
          }
          fmt.Println("写入成功:", putResp.Header.Revision)

          // 判断key是否过期
          for {
            if getResp, err = kv.Get(context.TODO(), "/cron/lock/job3"); err != nil {
              fmt.Println(err)
              return
            }
            // 如果等于0,说明过期了
            if getResp.Count == 0 {
              fmt.Println("kv过期了")
              break
            } else {
              fmt.Println("没过期", getResp.Kvs)
            }
            time.Sleep(2 * time.Second)
          }
          
          //TODO watch
    }
```

> * Revision  作用域为集群，逻辑时间戳，全局单调递增，任何 key 修改都会使其自增

> * CreateRevision   作用域为 key, 等于创建这个 key 时的 Revision, 直到删除前都保持不变

> * ModRevision 作用域为 key, 等于修改这个 key 时的 Revision, 只要这个 key 更新都会改变

> * Version 作用域为 key, 某一个 key 的修改次数(从创建到删除)，与以上三个 Revision 无关

关于 watch 哪个版本：

1、watch 某一个 key 时，想要从历史记录开始就用 CreateRevision，最新一条(这一条直接返回)开始就用 ModRevision

2、watch 某个前缀，就必须使用 Revision






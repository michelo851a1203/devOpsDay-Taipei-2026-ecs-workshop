# pulumi 的使用說明

(可以參考 `pulumi-deploy`) 裡面的專案  

操作方式 -> 記得安裝 pulumi 可以參考 `https://www.pulumi.com/docs/install/`

輸入以下指令，確認 pulumi 確認是否安裝完成
```sh
pulumi version
```

#### 使用

```sh
cd pulumi-deploy
```

- 初始化 pulumi stack
```sh
pulumi stack init
# stack name(dev) : dev
# Enter your passphrase to protect config/secrets : <自己記得住的密碼即可>
```

- 設定 pulumi config

```sh
pulumi config set aws:region ap-east-2 # 如果不是用 Taipei 可以用其他 region
pulumi config set aws:profile <profile_name> # 這裡可以輸入自己的 profile
```

- 檢查看看能否跑得起來

```sh
pulumi previw
```

- 開始部署

```sh
pulumi up
```

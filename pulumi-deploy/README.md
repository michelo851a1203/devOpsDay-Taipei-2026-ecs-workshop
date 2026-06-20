# 使用說明

## 開始
```sh
pulumi stack init
```
and then
```sh
# 選 dev
# 輸入密碼
```
## 確認
```sh
pulumi stack ls
```

## 設置

```sh
pulumi config set aws:region <自己習慣用的 region>
# pulumi config set aws:region ap-east-2
```

```sh
pulumi config set aws:profile <自己的 aws profile>
# pulumi config set aws:profile AdministratorAccess-687126124212
```

## 啟動

```sh
pulumi preview
```

## 部署

```sh
pulumi up
```

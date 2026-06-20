驗證一下

這裡驗證 container insight 的 namespace 是否為 ECS/ContainerInsights
```sh
aws cloudwatch list-metrics \
--namespace "ECS/ContainerInsights" \
--region ap-east-2 \
--profile <profile_name> | head -50
```

這個可以看一下自己的 cluster 設定是否合規
```
aws ecs describe-clusters \
  --clusters supportive-gorilla-15sf6h \
  --include SETTINGS \
  --region ap-east-2 \
  --profile AdministratorAccess-687126124212
```

force new deployment

```sh
aws ecs update-service \
  --cluster supportive-gorilla-15sf6h \
  --service api-green-service-vqduadf6 \
  --force-new-deployment \
  --region ap-east-2 \
  --profile AdministratorAccess-687126124212
```

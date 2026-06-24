# 如何使用 terraform 安裝
## 1. 先決條件

- 安裝 [Terraform](https://developer.hashicorp.com/terraform/install)（>= 1.5）
- 設定好 AWS 認證（其中一種）：
  - `aws configure sso` / 設定 profile，並在 `terraform.tfvars` 填 `aws_profile`

---

## 1. 操作步驟

### 1-1. 填入變數

```sh
cp terraform.tfvars.example terraform.tfvars
# 編輯 terraform.tfvars，填入 vpc_id / subnet_ids / ecr_uri / region / aws_profile
```

### 1.2 初始化一切
```sh
terraform init
```

第一次執行會下載 AWS / random provider。

### 1-3. 預覽

```sh
terraform plan
```

會列出將要建立的資源數量與內容，不會真的動到雲端。

### 1-4. 部署

```sh
terraform apply
# 看完 plan 後輸入 yes
```

### 1-5. 取得輸出

```sh
terraform output
terraform output albDnsName   # 拿 ALB DNS 丟瀏覽器測試
```

### 4-6. 銷毀

```sh
terraform destroy
```

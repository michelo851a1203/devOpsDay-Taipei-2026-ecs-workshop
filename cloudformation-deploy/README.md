# CloudFormation 部署範例（ECS Fargate 藍綠 Demo）

## 1. 與 Pulumi 行為差異（重要）

1. **IAM 角色「有就用、沒有才建」的邏輯**
   CloudFormation 沒有「查詢現有資源」的能力，所以改用參數控制：
   - `ExistingExecutionRoleArn` 留空 → 由本 stack 建立角色（預設行為）。
   - 帳號內**已經有** `ecsTaskExecutionRole` → 把該角色 ARN 填進這個參數，stack 就會沿用，
     避免名稱衝突（`AlreadyExists`）。

4. **建立需要 IAM 權限**
   因為會建立具名 IAM 角色，部署指令都帶 `--capabilities CAPABILITY_NAMED_IAM`。

## 2.設定參數（直接改 template.yaml）

打開 `template.yaml`，把最上方 `Parameters` 區塊裡每個 `Default` 改成你自己的值
（標了 `← 改成你的值` 的那幾行）。這對應 `main.go` 最上方那段被框起來的變數：

| Parameter（改它的 Default） | 說明 | 對應 Pulumi 變數 |
| --- | --- | --- |
| `VpcId` | VPC ID | `vpcID` |
| `SubnetA/B/C` | 三個子網路 | `subnetA/B/C` |
| `EcrImageUri` | 容器映像 URI | `ecrURI` |
| `CpuArchitecture` | `ARM64` 或 `X86_64`（已預設 `ARM64`） | `cpuArchitecture` |
| `BlueServiceName` / `GreenServiceName` | 服務名稱（已給預設值） | 同名變數 |
| `ExistingExecutionRoleArn` | 既有角色 ARN（留空則建立） | 對應 LookupRole 邏輯 |

---

## 3.完整操作教學

### 步驟 0：驗證模板語法（可選，等同 Pulumi 編譯期檢查）

```sh
aws cloudformation validate-template \
		--template-body template.yaml \
		REGION=ap-east-2 PROFILE=<myprofile>
# make validate REGION=ap-east-2 PROFILE=myprofile
```

### 步驟 1：預覽變更 —— 等效 `pulumi preview`

```sh
make preview STACK=ecs-bluegreen-demo REGION=ap-east-2 PROFILE=myprofile
```

這會建立一個 **change set 但不執行**，並把「將新增/修改/刪除哪些資源」印出來。
看完確認沒問題後，再進行下一步部署。

> 原理：`aws cloudformation deploy --no-execute-changeset` 會算出差異（diff），
> 等同 Pulumi 的 preview。CloudFormation 第一次部署時，change set 會顯示所有資源為「Add」。

### 步驟 2：部署 —— 等效 `pulumi up`

```sh
aws cloudformation deploy \
    --template-file template.yaml \
    --stack-name dev \
    --capabilities CAPABILITY_NAMED_IAM \
    REGION=ap-east-2 PROFILE=<myprofile>
make up STACK=dev REGION=ap-east-2 PROFILE=<myprofile>
```

`aws cloudformation deploy` 會自動判斷：
- stack 不存在 → 建立（`CREATE`）
- stack 已存在 → 以差異更新（`UPDATE`）

指令會等到 stack 進入 `CREATE_COMPLETE` / `UPDATE_COMPLETE` 才返回。

### 步驟 3：查看輸出 —— 等效 `pulumi stack output`

```sh
make outputs STACK=dev REGION=ap-east-2 PROFILE=myprofile
```

會以表格列出 `albDnsName`、`clusterArn` 等輸出值。
用瀏覽器打開 `albDnsName` 即可測試服務。

### 步驟 4：銷毀 —— 等效 `pulumi destroy`

```sh
aws cloudformation delete-stack \
    --stack-name dev \
    --profile <profile_name> --region ap-east-2
aws cloudformation wait stack-delete-complete \
    --stack-name dev \
    --profile <profile_name> --region ap-east-2
# make destroy STACK dev REGION=ap-east-2 PROFILE=myprofile

```

會 `delete-stack` 並等到刪除完成。

---

## 七、不想用 Makefile？直接用 AWS CLI

> 因為參數值都寫在 `template.yaml` 的 `Default` 裡，這裡完全不需要 `--parameter-overrides`。

```sh
# 預覽 (pulumi preview)
aws cloudformation deploy \
  --template-file template.yaml \
  --stack-name ecs-bluegreen-demo \
  --capabilities CAPABILITY_NAMED_IAM \
  --no-execute-changeset \
  --region ap-east-2 --profile myprofile

# 部署 (pulumi up) —— 把上面那行的 --no-execute-changeset 拿掉即可
aws cloudformation deploy \
  --template-file template.yaml \
  --stack-name ecs-bluegreen-demo \
  --capabilities CAPABILITY_NAMED_IAM \
  --region ap-east-2 --profile myprofile

# 銷毀 (pulumi destroy)
aws cloudformation delete-stack \
  --stack-name ecs-bluegreen-demo \
  --region ap-east-2 --profile myprofile
```

---

## 八、排錯

部署失敗時，先看事件，找出第一個 `*_FAILED` 的資源與原因：

```sh
make events STACK=ecs-bluegreen-demo REGION=ap-east-2 PROFILE=myprofile
```

常見問題：

- **`ecsTaskExecutionRole already exists`**
  帳號已有同名角色 → 在 `template.yaml` 把該角色 ARN 填到 `ExistingExecutionRoleArn` 的 `Default`，再重跑 `make up`。
- **`demo-sg` / `demo-tg` / `demo-alb` already exists**
  這些名稱寫死了；若同 region 已存在同名資源，請先刪除舊資源或改名。
- **Service 卡在 `CREATE_IN_PROGRESS` 很久後失敗**
  通常是 task 起不來（映像拉取失敗、健康檢查 `/health` 失敗、subnet 沒有對外路由）。
  到 ECS Console 看 task 的 `Stopped reason` 與 CloudWatch log group `/ecs/sample-log`。

---

## 九、檔案結構

```
cloudformation-deploy/
├── template.yaml   # CloudFormation 模板（核心）— 參數值直接寫在 Default
├── Makefile        # preview / up / destroy 等指令捷徑
└── README.md       # 本說明
```

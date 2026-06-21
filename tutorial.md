# AWS Console 操作流程


## 把 app 送去 ECR

1. 去左上角的 **Search** 找 ECR(Elastic Container Registry)  

2. [如果沒創建過的話]這時會看到頁面上有 Create a repository -> 點選橘色按鈕 **Create**

3. 這時候看到 **General settings** 上方的卡片區域 輸入 **Repository name** : demo-arm

4. 這時不用動其他的設定，點選右下角的橘色按鈕 **Create**

5. 這時候會看到列表上有剛才創建的 `demo-arm`

6. 點選上面的 `demo-arm` 會跳到我們剛才創建的 repo 內

7. 這時候會看到右上方有橘色按鈕 **view push commands**

8. 這時候會有四個指令可以點選左邊的小方框複製

注意 : 複製完後在自己的 terminal 直接貼上後請不要直接 enter

-------------------------------------------
接下來會從 ECR 頁面複製指令，在自己的 terminal 上貼上，

!! 請留意 !!  terminal 的路徑請在 `workshop-sample`

如果現在你在 clone 的專案上，可以前往到 workshop-sample 目錄下(像是底下的指令，可以到該目錄下)

```sh
cd workshop-sample
```
-------------------------------------------

9. 先複製第一個指令，大概長得像這樣 

```sh
aws ecr get-login-password --region ap-east-2 | docker login --username AWS --password-stdin <acocunt_id>.dkr.ecr.ap-east-2.amazonaws.com
```

這裡我們要在 `|` 加上自己的 profile

如下

```sh
aws ecr get-login-password --region ap-east-2 --profile <profile_name> | docker login --username AWS --password-stdin <acocunt_id>.dkr.ecr.ap-east-2.amazonaws.com
```

這時候 有出現 `Login Success` 就代表第一個指令 okay 了

10. 這時候複製第二個指令，大概是長這樣的

```sh
docker build -t demo-arm .
```

這個是 docker 打包 image 的指令，請不要急著按 enter

我們這裡要做幾件事，如果後面的 os 想用 x86_64 的 指令請用


```sh
docker build --platform=linux/amd64 -t demo-arm .
```

如果想用 arm 的請用以下指令

```sh
docker build --platform=linux/arm64 -t demo-arm .
```

11. 如果打包好了後，可以用

```sh
docker images
```

確認是否在上面 有 demo-arm 的 image 名稱，如果有的話代表成功了

12. 如果第二個指令成功後，就往第三個指令，像是下面這樣

```sh
docker tag demo-arm:latest <account_id>.dkr.ecr.ap-east-2.amazonaws.com/demo-arm:latest
```

這時候第二個指令成功後，就直接複製貼上 enter 就好

13. 這時候要準備 push 上去了，我們複製第四個指令，像是以下這樣

```sh
docker push <acocunt_id>.dkr.ecr.ap-east-2.amazonaws.com/demo-arm:latest
```
這時候也依樣直接複製貼上 enter 就好，如果 push 成功後，可以回到自己 AWS ECR畫面

> 來到自己 ECR 的頁面可以重整，看到列表上會有三個項目在上面，代表這個步驟就成功了

## 設定 Security Group

1. 左上角 **Search** -> 尋找 **EC2**

2. 在左側的 SideBar (Network & Security)點選 **Security Group**

3. 點選右上角的橘色按鈕 ** Create Security group**

- Basic dtails
**Security group name** : demo-sg
**Description** : demo-sg

**VPC** 帶入預設

- Inbound rules
Type: HTTP (80)
Source: Anywhere Ipv4

- Outbound rules 不用動

4. 再創建一個 Security group

- Basic dtails
**Security group name** : demo-ecs-sg
**Description** : demo-ecs-sg

**VPC** 帶入預設

- Inbound rules
Type: Custom TCP
Port range: 8080
Source: 點選 input 找到剛才創建的 **demo-sg**

- Outbound rules 不用動

## 設定 TargetGroup

### Step 1 of 3:

1. 在左側 SideBar(Load Balancing) 點選 **Target Groups**

2. 右上角橘色按鈕點選 **Create target group**

- Settings:

**Target type** : Ip address
**Target group name** : demo-tg
**Protocol** : HTTP
**Port** : 8080 

Health checks : (原則上不用設定)

**Health check protocol** : HTTP
**Health check path** : /health

點選右下角的橘色按鈕**Next**

### Step 2 of 3 :

(Register targets)確認畫面資訊 okay
再點選右下角的橘色按鈕**Next**

### Step 3 of 3 :

Review and create
點選右下角的橘色按鈕**Create target group**

## 設定 Load Balancers (ALB)

1. 在左側 SideBar(Load Balancing) 點選 **Load Balancers**

2. 點選右上角 **Create load balancer**

3. 選擇 **Application Load Balancer**，點選下方的 **Create**

**Load balancer name** : demo-alb

- Network mapping

**Availability Zone and subnets**: 都選起來

- Security groups
**Security groups** : demo-sg

- Listeners and routing
**Protocol** : HTTP
**Port** : 80
**Target group**: demo-tg

這時候點選右下角的橘色按鈕**Create load balancer**

## step4 : 設定 AWS Cloud Map

1. 左上角 **Search** -> 尋找 **AWS Cloud map**

2. 點選橘色按鈕 **Create namespace**

**Namespace name** : workshop.local

**Instance discovery** : API call (基本上不用改)

點選右下角橘色按鈕 **Create namespace**

## 設定 ECS - Cluster

1. 左上角 **Search** -> 尋找 **Elastic Container Servic**

2. 在左側 SideBar 點選 **Cluster**

3. 點選右上角的橘色按鈕 **Create cluster**

- Infrastructure 
Fargate only
(基本上不用動任何設定)

- Monitoring - optional

**Select the level of observability you want to achieve with Container Insights**
點選 **Container Insight**

4. 點選右下角的橘色按鈕**Create**

## step6 : 設定 Task definition (api-blue)

1. 在左側 SideBar 點選 **Task deinfitions**

Create new task definition

**Task definition family** : api-blue

- Infrastructure requirements:

**Launch type**: AWS Fargate

**Operating system/Architecture**: Linux/ARM64 (如果 docker build 是用 linux/amd64 則選 linux/x86_64)
**CPU** : .5vCPU
**Memory** : 1GB

**Task execution role** : ecsTaskExecutionRole(或是 create new role 如果第一次創建的話)

- Container-1

**Name** : api
**image URI** : 選擇我們剛創建的 demo-arm : (Type : Image,且有 Image Size)
**Container port** : 8080
**Protocol** : TCP
**Port name** : api-port

- Environment variables - optional

**APP_VERSION** : blue
**FAIL_RATE** : 0

- Log Collection

**awslogs-group** -> value : /ecs/sample-log
**awslogs-region** -> value : <你所在的區域>
**awslogs-stream-prefix** -> value : api-blue

## 設定 Task definition (api-green)

1. 在左側 SideBar 點選 **Task deinfitions**

**Task definition family** : api-green

- Infrastructure requirements:

**Launch type**: AWS Fargate

**Operating system/Architecture**: Linux/ARM64 (如果 docker build 是用 linux/amd64 則選 linux/x86_64)
**CPU** : .5vCPU
**Memory** : 1GB

**Task execution role** : ecsTaskExecutionRole(或是 create new role 如果第一次創建的話)

- Container-1

**Name** : api
**image URI** : 選擇我們剛創建的 demo-arm : (Type : Image,且有 Image Size)
**Container port** : 8080
**Protocol** : TCP
**Port name** : api-port


- Environment variables - optional

**APP_VERSION** : green
**FAIL_RATE** : 0

- Log Collection

**awslogs-group** -> value : /ecs/sample-log
**awslogs-region** -> value : <你所在的區域>
**awslogs-stream-prefix** -> value : api-green

## 調整 剛才創建的 Task execution role 權限 (通常是 ecsTaskExecutionRole)

1. 左上角 **Search** -> 尋找 **iam**

2. 在左側 SideBar(Access Management) 點選 **Roles**

3. 在上方的搜巡欄找<剛才 Task execution role>的名字(通常是 ecsTaskExecutionRole)

4. 在 **Permissions** 的頁籤 **Permissions policie**
點選 Add permissions 的按鈕 -> Attach policy

找到 CloudWatchLogsFullAccess(可在上方搜尋匡找) 
並且點擊下方按鈕 **Add permissions**

就可以關上 iam 了

## step 8 : 創建服務(api-blue)

1. 回到 **ECS**

2. 在左側 SideBar(Access Management) 點選 **Clusters**

3. 頁籤**Services** 點選右邊或下方的橘色按鈕 **Create**

4. **Task definition family** : 選擇 api-blue

- Deployment configuration:

**Desired tasks** : 2

- Networking

**Security group name** : demo-ecs-sg

- Service Connect

✅Use Service Connect

**Service Connect configuration** : 選擇 Client and server

**Namespace** : 選擇 `workshop.local`

點選 **Add port mappings and application**

**Port alias** : api-port
**Discovery** : api-blue
**DNS** : api-blue
**Port** : 8080

- use log collection
**awslogs-group** : /ecs/sample-log
**awslogs-stream-prefix** : api-blue

- Load balancing

✅Use Load balancing

**Application Load Balancer** : Use an existing load balancer
**Load balancer** : demo-alb

**Listener**: Use an existing listener
HTTP:80

**Target group** : Use an existing target group
demo-tg

## 創建服務(api-green)

1. 回到 **ECS**

2. 在左側 SideBar(Access Management) 點選 **Clusters**

3. 頁籤**Services** 點選右邊或下方的橘色按鈕 **Create**

4. **Task definition family** : 選擇 api-green

- Deployment configuration:

**Desired tasks** : 1

- Networking

**Security group name** : demo-ecs-sg

- Service Connect

✅Use Service Connect

**Service Connect configuration** : 選擇 Client and server

**Namespace** : 選擇 `workshop.local`

點選 **Add port mappings and application**

**Port alias** : api-port
**Discovery** : api-green
**DNS** : api-green
**Port** : 8080

- use log collection
**awslogs-group** : /ecs/sample-log
**awslogs-stream-prefix** : api-green

- Load balancing

✅Use Load balancing

**Application Load Balancer** : Use an existing load balancer
**Load balancer** : demo-alb

**Listener**: Use an existing listener
HTTP:80

**Target group** : Use an existing target group
demo-tg

## 檢查

1. 回到 **EC2**

2. 在左側 SideBar(Load Balancers) 點選 **demo-alb**

3. 複製 **DNS name**

4. 用 curl 或 瀏覽器觀看結果


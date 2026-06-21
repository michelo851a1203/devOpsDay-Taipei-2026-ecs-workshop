# 如何把 app 送去 ECR

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

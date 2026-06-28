# 🚀 DevOps Day Taipei 2026
### Go 整合 ECS Service Connect 與 LLM 實作 AIOps 自動化決策

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)
![AWS ECS](https://img.shields.io/badge/AWS-ECS-FF9900?logo=amazonaws&logoColor=white)
![Pulumi](https://img.shields.io/badge/IaC-Pulumi%20%7C%20Terraform%20%7C%20CloudFormation-844FBA)

## 📑 目錄
- [大綱](#大綱)
- [快速開始](#快速開始)
- [部署方式](#部署方式)
- [教學文件](#教學文件)

## 大綱

在現代雲端環境中，金絲雀發布 (Canary Deployment) 的難點不再是流量切換，
而是「何時該切換」與「何時該回滾」。
本工作坊將跳過基礎概念，直接進入開發者視角，
利用 Golang 與 ECS Service Connect 實作一套具備 AI 判定能力的自動化系統。

數據提取與分析 ： 
使用 Golang 整合 CloudWatch Container Insights 與 ECS Service Connect 原生遙測資料，
即時監控新版本服務的健康狀態，包含 Request Rate、Error Rate 與 Connection 級別的 Latency。

AI 決策大腦 ： 實作 Golang 與 AWS Bedrock (LLM) 的介接，
讓 AI 根據 Error Rate 與 Latency 趨勢給出具備邏輯判斷的部署指令，
並產出可審計的決策理由鏈 (Chain-of-Thought)。

基礎設施聯動 ： 深入 AWS SDK v2 for Go，
透過 ECS API 動態調整 Blue/Green Task 的 Desired Count 與 Service 權重，
搭配 Service Connect 的 Namespace 路由機制，達成無縫自動回滾。

混沌演練 ： 
現場注入應用程式故障，
觀察系統如何在無人介入下，
透過 AIOps 決策迴路完成智慧型故障隔離與流量收斂。

選擇自己喜歡的方式把服務起起來  


## 🛠 部署方式

> 💡 記得先把 `workshop-sample` 推上 ECR，可參考 [ecr_push_tutorial.md](./ecr_push_tutorial.md)

| 方式 | 適用情境 | 教學文件 |
|------|----------|----------|
| AWS Console | 手動部署、初學者 | [tutorial.md](./tutorial.md) / [tutorial-full.md](./tutorial-full.md) |
| Pulumi | 偏好程式語言寫 IaC | [pulumi_tutorial.md](./pulumi_tutorial.md) |
| Terraform | 業界主流 IaC | [terraform-deploy/](./terraform-deploy) |
| CloudFormation | AWS 原生 IaC | [cloudformation-deploy/](./cloudformation-deploy) |


## 📄 License
本專案採用 [MIT License](./LICENSE) 授權。

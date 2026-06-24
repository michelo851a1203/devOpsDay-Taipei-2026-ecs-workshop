# ===========================================================================
# CloudFormation 操作捷徑 —— 對應 Pulumi 的 preview / up / destroy
# ===========================================================================
# 用法範例:
#   make preview STACK=ecs-bluegreen-demo REGION=ap-east-2 PROFILE=myprofile
#   make up      STACK=ecs-bluegreen-demo REGION=ap-east-2 PROFILE=myprofile
#   make destroy STACK=ecs-bluegreen-demo REGION=ap-east-2 PROFILE=myprofile
# ===========================================================================

STACK    ?= ecs-bluegreen-demo
TEMPLATE ?= template.yaml
REGION   ?=
PROFILE  ?=

# 把 REGION / PROFILE 組成 aws CLI 旗標 (留空就不加)
AWS_FLAGS  = $(if $(REGION),--region $(REGION),) $(if $(PROFILE),--profile $(PROFILE),)

# 教學版：所有參數值都寫死在 template.yaml 的 Default 裡，
# 不需要 params.json，也不需要 python，故沒有 --parameter-overrides。

.PHONY: help validate preview up destroy outputs events

help:
	@echo "make validate  -> 驗證模板語法 (等同 pulumi 編譯期檢查)"
	@echo "make preview   -> 產生並顯示 change set，不執行 (等同 pulumi preview)"
	@echo "make up        -> 建立/更新 stack (等同 pulumi up)"
	@echo "make destroy   -> 刪除 stack (等同 pulumi destroy)"
	@echo "make outputs   -> 顯示 stack 的 Outputs (等同 pulumi stack output)"
	@echo "make events    -> 顯示最近的部署事件"

# --- 語法驗證 -------------------------------------------------------------
validate:
	aws cloudformation validate-template \
		--template-body file://$(TEMPLATE) \
		$(AWS_FLAGS)

# --- pulumi preview 等效: 建立 change set 但不執行 ------------------------
preview:
	aws cloudformation deploy \
		--template-file $(TEMPLATE) \
		--stack-name $(STACK) \
		--capabilities CAPABILITY_NAMED_IAM \
		--no-execute-changeset \
		$(AWS_FLAGS)

# --- pulumi up 等效: 建立或更新 stack ------------------------------------
up:
	aws cloudformation deploy \
		--template-file $(TEMPLATE) \
		--stack-name $(STACK) \
		--capabilities CAPABILITY_NAMED_IAM \
		$(AWS_FLAGS)

# --- pulumi destroy 等效: 刪除 stack -------------------------------------
destroy:
	aws cloudformation delete-stack \
		--stack-name $(STACK) \
		$(AWS_FLAGS)
	aws cloudformation wait stack-delete-complete \
		--stack-name $(STACK) \
		$(AWS_FLAGS)

# --- 查看 Outputs --------------------------------------------------------
outputs:
	aws cloudformation describe-stacks \
		--stack-name $(STACK) \
		--query "Stacks[0].Outputs" \
		--output table \
		$(AWS_FLAGS)

# --- 查看部署事件 (排錯用) -----------------------------------------------
events:
	aws cloudformation describe-stack-events \
		--stack-name $(STACK) \
		--max-items 25 \
		--query "StackEvents[].{Time:Timestamp,Status:ResourceStatus,Type:ResourceType,Reason:ResourceStatusReason}" \
		--output table \
		$(AWS_FLAGS)

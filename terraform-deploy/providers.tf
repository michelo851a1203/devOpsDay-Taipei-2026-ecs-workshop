provider "aws" {
  region = var.region
  # 對應 pulumi config set aws:profile <profile>
  # 若使用環境變數 / 預設 profile，可將 profile 留空字串
  profile = var.aws_profile != "" ? var.aws_profile : null
}

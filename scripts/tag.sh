#!/usr/bin/env bash
# tag.sh - 自动生成下一个 git tag 版本
# 用法:
#   ./tag.sh          # patch +1 (默认)
#   ./tag.sh -M       # major +1
#   ./tag.sh -m       # minor +1
#   ./tag.sh -p       # patch +1 (显式)
# 创建后自动推送到远端

set -e

BUMP="patch"

for arg in "$@"; do
  case $arg in
    -M|--major) BUMP="major" ;;
    -m|--minor) BUMP="minor" ;;
    -p|--patch) BUMP="patch" ;;
    *)
      echo "未知参数: $arg"
      echo "用法: $0 [-M|-m|-p]"
      exit 1
      ;;
  esac
done

# 获取最新 tag（按版本号排序）
LATEST=$(git tag --sort=-v:refname | grep -E '^v?[0-9]+\.[0-9]+\.[0-9]+$' | head -1)

if [[ -z "$LATEST" ]]; then
  echo "未找到符合 vX.Y.Z 格式的 tag，从 v0.0.1 开始"
  NEW_TAG="v0.0.1"
else
  echo "当前最新 tag: $LATEST"

  # 去掉前缀 v
  VERSION="${LATEST#v}"
  MAJOR=$(echo "$VERSION" | cut -d. -f1)
  MINOR=$(echo "$VERSION" | cut -d. -f2)
  PATCH=$(echo "$VERSION" | cut -d. -f3)

  case $BUMP in
    major)
      MAJOR=$((MAJOR + 1))
      MINOR=0
      PATCH=0
      ;;
    minor)
      MINOR=$((MINOR + 1))
      PATCH=0
      ;;
    patch)
      PATCH=$((PATCH + 1))
      ;;
  esac

  NEW_TAG="v${MAJOR}.${MINOR}.${PATCH}"
fi

echo "新 tag: $NEW_TAG  (bump: $BUMP)"
read -r -p "确认创建? [Y/n] " CONFIRM
CONFIRM="${CONFIRM:-Y}"

if [[ "$CONFIRM" =~ ^[Yy]$ ]]; then
  git tag "$NEW_TAG"
  echo "已创建 tag: $NEW_TAG"
  git push origin "$NEW_TAG"
  echo "已推送 tag: $NEW_TAG"
else
  echo "已取消"
fi

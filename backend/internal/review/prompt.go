package review

import "fmt"

func BuildReviewPrompt(diffContent string) string {
	return fmt.Sprintf(`你是资深代码审查专家。请 Review 以下代码变更，评估:
1. 代码质量 (可读性、可维护性)
2. 安全性 (注入、XSS 等)
3. 错误处理 (是否妥善处理异常)
4. 代码风格 (是否符合项目规范)
5. 测试覆盖 (是否有足够测试)

代码变更 (diff):
%s

输出严格 JSON 格式:
{
  "summary": "总体评价",
  "issues": [
    {
      "severity": "error|warning|info",
      "file": "文件路径",
      "line": 行号,
      "code_snippet": "相关代码片段",
      "message": "问题描述",
      "suggestion": "修改建议"
    }
  ],
  "categories": {
    "security": {"status": "passed|warning|failed", "details": "说明"},
    "error_handling": {"status": "passed|warning|failed", "details": "说明"},
    "code_style": {"status": "passed|warning|failed", "details": "说明"},
    "test_coverage": {"status": "passed|warning|failed", "details": "说明"}
  }
}
只输出 JSON，不要任何其他内容。`, diffContent)
}

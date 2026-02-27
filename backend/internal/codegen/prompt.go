package codegen

import (
	"fmt"
	"strings"

	"github.com/codeMaster/backend/internal/model"
)

type PromptInput struct {
	RepoAnalysis *model.AnalysisResult
	Requirement  *model.Requirement
	ExtraContext string
	DocContent   string
}

func BuildPrompt(input PromptInput) string {
	var sb strings.Builder

	sb.WriteString("你是一位资深软件工程师，正在为一个实际项目编写代码。请严格遵循项目现有的技术栈和代码风格。\n\n")

	if input.RepoAnalysis != nil {
		sb.WriteString("## 项目上下文\n\n")
		if len(input.RepoAnalysis.TechStack) > 0 {
			sb.WriteString(fmt.Sprintf("技术栈: %s\n", strings.Join(input.RepoAnalysis.TechStack, ", ")))
		}
		if input.RepoAnalysis.DirectoryStructure != "" {
			sb.WriteString(fmt.Sprintf("项目结构: %s\n", input.RepoAnalysis.DirectoryStructure))
		}
		if input.RepoAnalysis.CodeStyle.Naming != "" {
			sb.WriteString(fmt.Sprintf("命名规范: %s\n", input.RepoAnalysis.CodeStyle.Naming))
		}
		if input.RepoAnalysis.CodeStyle.ErrorHandling != "" {
			sb.WriteString(fmt.Sprintf("错误处理: %s\n", input.RepoAnalysis.CodeStyle.ErrorHandling))
		}
		if input.RepoAnalysis.CodeStyle.TestFramework != "" {
			sb.WriteString(fmt.Sprintf("测试框架: %s\n", input.RepoAnalysis.CodeStyle.TestFramework))
		}
		if len(input.RepoAnalysis.Modules) > 0 {
			sb.WriteString("\n模块说明:\n")
			for _, m := range input.RepoAnalysis.Modules {
				sb.WriteString(fmt.Sprintf("- %s: %s (%d 文件)\n", m.Path, m.Description, m.FilesCount))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## 需求\n\n")
	sb.WriteString(fmt.Sprintf("### %s\n\n", input.Requirement.Title))
	sb.WriteString(input.Requirement.Description)
	sb.WriteString("\n\n")

	if input.DocContent != "" {
		sb.WriteString("## 关联文档内容\n\n")
		sb.WriteString(input.DocContent)
		sb.WriteString("\n\n")
	}

	if input.ExtraContext != "" {
		sb.WriteString("## 补充说明\n\n")
		sb.WriteString(input.ExtraContext)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## 编码要求\n\n")
	sb.WriteString("1. 在编写代码前，先阅读相关现有文件了解项目结构和风格\n")
	sb.WriteString("2. 严格遵循项目现有代码风格（命名、目录结构、错误处理方式）\n")
	sb.WriteString("3. 为新增功能编写单元测试\n")
	sb.WriteString("4. 完成编码后执行编译/构建命令确保无语法错误\n")
	sb.WriteString("5. 不要修改与需求无关的文件\n")

	return sb.String()
}

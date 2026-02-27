import { useNavigate } from 'react-router-dom';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import {
  BookOpen,
  FolderKanban,
  GitBranch,
  FileText,
  Zap,
  ClipboardCheck,
  GitMerge,
  ArrowRight,
  Settings,
  KeyRound,
} from 'lucide-react';

const workflowSteps = [
  {
    icon: FolderKanban,
    title: '创建项目',
    description: '创建一个新项目，作为需求和代码管理的基本单元。每个项目可以关联多个代码仓库。',
    link: '/projects',
    linkLabel: '前往项目列表',
    color: 'text-blue-500',
    bg: 'bg-blue-500/10',
  },
  {
    icon: GitBranch,
    title: '添加仓库并分析',
    description: '在项目中关联 Git 仓库，平台会自动分析仓库结构，为后续 AI 代码生成提供上下文。',
    link: '/projects',
    linkLabel: '管理仓库',
    color: 'text-cyan-500',
    bg: 'bg-cyan-500/10',
  },
  {
    icon: FileText,
    title: '创建需求',
    description: '编写需求描述，包括功能说明、验收标准等。支持 Markdown 格式，可以关联项目和仓库。',
    link: '/requirements',
    linkLabel: '前往需求列表',
    color: 'text-orange-500',
    bg: 'bg-orange-500/10',
  },
  {
    icon: Zap,
    title: 'AI 代码生成',
    description: '基于需求描述和仓库上下文，AI 自动生成代码变更。可以查看生成进度和实时日志。',
    link: '/requirements',
    linkLabel: '查看需求',
    color: 'text-green-500',
    bg: 'bg-green-500/10',
  },
  {
    icon: ClipboardCheck,
    title: 'AI 审查 + 人工审查',
    description: 'AI 先进行自动代码审查并打分，然后由团队成员进行人工审查，确保代码质量。',
    link: '/reviews',
    linkLabel: '前往审查列表',
    color: 'text-purple-500',
    bg: 'bg-purple-500/10',
  },
  {
    icon: GitMerge,
    title: '创建合并请求',
    description: '审查通过后，一键创建合并请求（Merge Request），将生成的代码合并到目标分支。',
    link: '/reviews',
    linkLabel: '查看审查',
    color: 'text-pink-500',
    bg: 'bg-pink-500/10',
  },
];

export function GuidePage() {
  const navigate = useNavigate();

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <BookOpen className="w-6 h-6 text-primary" />
          使用指南
        </h1>
        <p className="text-muted-foreground mt-1">了解 CodeMaster 的核心工作流程，快速上手</p>
      </div>

      {/* Platform Intro */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">平台简介</CardTitle>
          <CardDescription>
            CodeMaster 是一个 AI 驱动的代码生成与管理平台。通过自然语言描述需求，AI
            自动生成代码并进行智能审查，帮助团队提升开发效率和代码质量。
          </CardDescription>
        </CardHeader>
      </Card>

      {/* Workflow Steps */}
      <div>
        <h2 className="text-lg font-semibold mb-4">核心工作流</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {workflowSteps.map((step, index) => {
            const Icon = step.icon;
            return (
              <Card key={step.title} className="flex flex-col">
                <CardHeader className="pb-3">
                  <div className="flex items-center gap-3">
                    <div className={`w-10 h-10 rounded-lg ${step.bg} flex items-center justify-center ${step.color}`}>
                      <Icon className="w-5 h-5" />
                    </div>
                    <div className="flex items-center gap-2">
                      <Badge variant="outline" className="text-xs">
                        步骤 {index + 1}
                      </Badge>
                      <CardTitle className="text-base">{step.title}</CardTitle>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="flex-1 flex flex-col justify-between gap-4">
                  <p className="text-sm text-muted-foreground">{step.description}</p>
                  <Button variant="outline" size="sm" className="w-fit" onClick={() => navigate(step.link)}>
                    {step.linkLabel}
                    <ArrowRight className="w-3.5 h-3.5 ml-1" />
                  </Button>
                </CardContent>
              </Card>
            );
          })}
        </div>
      </div>

      <Separator />

      {/* Settings Tip */}
      <Card className="border-dashed">
        <CardContent className="p-6">
          <div className="flex items-start gap-4">
            <div className="w-10 h-10 rounded-lg bg-muted flex items-center justify-center text-muted-foreground">
              <KeyRound className="w-5 h-5" />
            </div>
            <div className="flex-1">
              <h3 className="font-medium mb-1">开始之前</h3>
              <p className="text-sm text-muted-foreground mb-3">
                使用平台前，请先在设置页面配置 AI 模型的 API Key 和 Git 仓库的访问 Token，以确保所有功能正常工作。
              </p>
              <Button variant="outline" size="sm" onClick={() => navigate('/settings')}>
                <Settings className="w-3.5 h-3.5 mr-1" />
                前往设置
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

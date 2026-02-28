import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { requirementApi } from '@/api/requirement';
import { projectApi } from '@/api/project';
import { repositoryApi } from '@/api/repository';
import { codegenApi } from '@/api/codegen';
import { feishuApi } from '@/api/feishu';
import { useAuth } from '@/hooks/use-auth';
import { useToast } from '@/hooks/use-toast';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import { Loading, Spinner } from '@/components/ui/spinner';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from '@/components/ui/dialog';
import { DeleteConfirmDialog } from '@/components/DeleteConfirmDialog';
import {
  FileText, ArrowLeft, Edit3, Play, Clock, CheckCircle, XCircle,
  AlertCircle, ExternalLink, Trash2, ArrowRight, Plus, Loader2,
  ClipboardCheck, Users, GitCompare, GitBranch, Copy, Upload, AlertTriangle,
} from 'lucide-react';
import type { Requirement, CodeGenTask, ProjectMember, Repository, DocLink } from '@/types';

const statusLabel: Record<string, string> = {
  draft: '草稿', generating: '生成中', generated: '已生成',
  reviewing: '审查中', approved: '已通过', merged: '已合并', rejected: '已拒绝',
};

const statusVariant = (status: string) => {
  switch (status) {
    case 'draft': return 'secondary' as const;
    case 'generating': return 'warning' as const;
    case 'generated': return 'default' as const;
    case 'reviewing': return 'outline' as const;
    case 'approved': case 'merged': return 'success' as const;
    case 'rejected': return 'destructive' as const;
    default: return 'secondary' as const;
  }
};

const priorityVariant = (p: string) => {
  if (p === 'p0') return 'destructive' as const;
  if (p === 'p1') return 'warning' as const;
  return 'secondary' as const;
};

function buildCompareUrl(gitUrl: string, platform: string, sourceBranch: string, targetBranch: string): string {
  const base = gitUrl.replace(/\.git$/, '');
  if (platform === 'github') {
    return `${base}/compare/${sourceBranch}...${targetBranch}`;
  }
  return `${base}/-/compare/${sourceBranch}...${targetBranch}`;
}

export function RequirementDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { user, isAdmin } = useAuth();
  const { toast } = useToast();
  const [req, setReq] = useState<Requirement | null>(null);
  const [loading, setLoading] = useState(true);
  const [editOpen, setEditOpen] = useState(false);
  const [editData, setEditData] = useState({ title: '', description: '', priority: '', assignee_id: '', repository_id: '', deadline: '' });
  const [editDocLinks, setEditDocLinks] = useState<DocLink[]>([]);
  const [editDocResolving, setEditDocResolving] = useState<Record<number, boolean>>({});
  const [editDocErrors, setEditDocErrors] = useState<Record<number, string>>({});
  const [editLoading, setEditLoading] = useState(false);
  const [members, setMembers] = useState<ProjectMember[]>([]);
  const [repos, setRepos] = useState<Repository[]>([]);
  const [generateOpen, setGenerateOpen] = useState(false);
  const [generateLoading, setGenerateLoading] = useState(false);
  const [extraContext, setExtraContext] = useState('');
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [reviewDialogOpen, setReviewDialogOpen] = useState(false);
  const [selectedReviewers, setSelectedReviewers] = useState<number[]>([]);
  const [triggeringReview, setTriggeringReview] = useState(false);
  const [manualSubmitOpen, setManualSubmitOpen] = useState(false);
  const [manualSubmitLoading, setManualSubmitLoading] = useState(false);
  const [manualCommitMessage, setManualCommitMessage] = useState('');
  const [manualCommitUrl, setManualCommitUrl] = useState('');

  const fetchRequirement = async () => {
    if (!id) return;
    try {
      const data = await requirementApi.get(Number(id));
      setReq(data);
    } catch (err) {
      toast({ title: '获取需求失败', description: (err as Error).message, variant: 'destructive' });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchRequirement(); }, [id]);

  useEffect(() => {
    if (!req?.project?.id) return;
    const projectId = req.project.id;
    projectApi.get(projectId).then((p) => setMembers(p.members ?? [])).catch(() => {});
    repositoryApi.listByProject(projectId, { page_size: 100 }).then((d) => setRepos(d.list)).catch(() => {});
  }, [req?.project?.id]);

  if (loading) return <Loading />;
  if (!req) return <div className="p-6 text-center text-muted-foreground">需求不存在</div>;

  const isCreator = req.creator.id === user?.id;
  const isAssignee = req.assignee?.id === user?.id;
  const canEdit = (isCreator || isAdmin) && (req.status === 'draft' || req.status === 'rejected');
  const canGenerate = (isAssignee || isAdmin) && req.status !== 'generating' && req.repository && req.assignee;

  const handleEdit = async () => {
    setEditLoading(true);
    try {
      const data: Record<string, unknown> = {
        title: editData.title,
        description: editData.description,
        priority: editData.priority,
      };
      if (editData.assignee_id) data.assignee_id = Number(editData.assignee_id);
      if (editData.repository_id) data.repository_id = Number(editData.repository_id);
      if (editData.deadline) data.deadline = new Date(editData.deadline + 'T23:59:59').toISOString();
      const validLinks = editDocLinks
        .filter((l) => l.url.trim())
        .map((l) => ({ ...l, title: l.title.trim() || l.url.trim() }));
      data.doc_links = validLinks;
      await requirementApi.update(req.id, data);
      toast({ title: '需求已更新', variant: 'success' });
      setEditOpen(false);
      fetchRequirement();
    } catch (err) {
      toast({ title: '更新失败', description: (err as Error).message, variant: 'destructive' });
    } finally {
      setEditLoading(false);
    }
  };

  const handleGenerate = async () => {
    setGenerateLoading(true);
    try {
      const result = await requirementApi.generate(req.id, { extra_context: extraContext || undefined });
      toast({ title: '代码生成已启动', variant: 'success' });
      setGenerateOpen(false);
      navigate(`/codegen/${result.task_id}`);
    } catch (err) {
      toast({ title: '启动失败', description: (err as Error).message, variant: 'destructive' });
    } finally {
      setGenerateLoading(false);
    }
  };

  const handleManualSubmit = async () => {
    if (!manualCommitUrl.trim()) {
      toast({ title: '请输入 Commit 链接', variant: 'destructive' });
      return;
    }
    setManualSubmitLoading(true);
    try {
      await requirementApi.manualSubmit(req.id, {
        commit_message: manualCommitMessage || undefined,
        commit_url: manualCommitUrl,
      });
      toast({ title: '手动提交成功', variant: 'success' });
      setManualSubmitOpen(false);
      setManualCommitMessage('');
      setManualCommitUrl('');
      fetchRequirement();
    } catch (err) {
      toast({ title: '提交失败', description: (err as Error).message, variant: 'destructive' });
    } finally {
      setManualSubmitLoading(false);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    toast({ title: '已复制到剪贴板', variant: 'success' });
  };

  const taskStatusIcon = (status: string) => {
    switch (status) {
      case 'completed': return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'failed': return <XCircle className="w-4 h-4 text-red-500" />;
      case 'cancelled': return <AlertCircle className="w-4 h-4 text-muted-foreground" />;
      case 'running': case 'cloning': return <Spinner size="sm" />;
      default: return <Clock className="w-4 h-4 text-muted-foreground" />;
    }
  };

  // Find latest completed task for diff link and review trigger
  const latestCompletedTask = req.codegen_tasks?.find((t) => t.status === 'completed') ?? null;

  const handleOpenReviewDialog = () => {
    setSelectedReviewers([]);
    setReviewDialogOpen(true);
  };

  const handleTriggerReview = async () => {
    if (!latestCompletedTask) return;
    setTriggeringReview(true);
    try {
      await codegenApi.triggerReview(latestCompletedTask.id, { reviewer_ids: selectedReviewers });
      toast({ title: 'Review 已发起', variant: 'success' });
      setReviewDialogOpen(false);
      fetchRequirement();
    } catch (err) {
      toast({ title: '发起 Review 失败', description: (err as Error).message, variant: 'destructive' });
    } finally {
      setTriggeringReview(false);
    }
  };

  const toggleReviewer = (id: number) => {
    setSelectedReviewers((prev) =>
      prev.includes(id) ? prev.filter((r) => r !== id) : [...prev, id]
    );
  };

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <Button variant="ghost" size="sm" className="mb-2" onClick={() => navigate(-1)}>
            <ArrowLeft className="w-4 h-4 mr-1" />返回
          </Button>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <FileText className="w-6 h-6 text-primary" />
            {req.title}
          </h1>
          <div className="flex items-center gap-2 mt-2">
            <Badge variant={statusVariant(req.status)}>{statusLabel[req.status] || req.status}</Badge>
            <Badge variant={priorityVariant(req.priority)}>{req.priority.toUpperCase()}</Badge>
            {req.deadline && new Date(req.deadline) < new Date() && req.status !== 'merged' && req.status !== 'approved' && (
              <Badge variant="destructive" className="gap-1">
                <AlertTriangle className="w-3 h-3" />已延期
              </Badge>
            )}
            {req.project && (
              <span className="text-sm text-muted-foreground cursor-pointer hover:text-foreground"
                onClick={() => navigate(`/projects/${req.project!.id}`)}>
                {req.project.name}
              </span>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          {latestCompletedTask && req.repository?.git_url && (
            <a
              href={buildCompareUrl(req.repository.git_url, req.repository.platform || 'gitlab', latestCompletedTask.source_branch, latestCompletedTask.target_branch)}
              target="_blank"
              rel="noopener noreferrer"
            >
              <Button variant="outline" size="sm">
                <GitCompare className="w-4 h-4 mr-1" />在 Git 中查看 Diff
              </Button>
            </a>
          )}
          {latestCompletedTask && (
            <Button variant="outline" size="sm" onClick={handleOpenReviewDialog}>
              <ClipboardCheck className="w-4 h-4 mr-1" />发起 Review
            </Button>
          )}
          {canEdit && (
            <>
              <Button variant="outline" size="sm" onClick={() => {
                setEditData({
                  title: req.title,
                  description: req.description,
                  priority: req.priority,
                  assignee_id: req.assignee ? String(req.assignee.id) : '',
                  repository_id: req.repository ? String(req.repository.id) : '',
                  deadline: req.deadline ? req.deadline.split('T')[0] : '',
                });
                setEditDocLinks(req.doc_links ? req.doc_links.map((l) => ({ ...l })) : []);
                setEditOpen(true);
              }}>
                <Edit3 className="w-4 h-4 mr-1" />编辑
              </Button>
              <Button variant="outline" size="sm" className="text-destructive" onClick={() => setDeleteOpen(true)}>
                <Trash2 className="w-4 h-4 mr-1" />删除
              </Button>
            </>
          )}
          {canGenerate && (
            <>
              <Button variant="outline" onClick={() => setManualSubmitOpen(true)}>
                <Upload className="w-4 h-4 mr-1" />手动提交
              </Button>
              <Button onClick={() => setGenerateOpen(true)}>
                <Play className="w-4 h-4 mr-1" />生成代码
              </Button>
            </>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Description */}
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader><CardTitle className="text-lg">需求描述</CardTitle></CardHeader>
            <CardContent>
              <div className="prose prose-sm dark:prose-invert max-w-none whitespace-pre-wrap">
                {req.description}
              </div>
              {req.doc_links && req.doc_links.length > 0 && (
                <div className="mt-4 pt-4 border-t">
                  <p className="text-sm font-medium mb-2">关联文档</p>
                  <div className="space-y-1.5">
                    {req.doc_links.map((link, i) => (
                      <a key={i} href={link.url} target="_blank" rel="noreferrer"
                        className="flex items-center gap-2 text-sm text-primary hover:underline py-1">
                        <ExternalLink className="w-3.5 h-3.5 shrink-0" />
                        <span>{link.title}</span>
                        {link.type && link.type !== 'other' && (
                          <Badge variant="secondary" className="text-[10px] px-1.5 py-0">
                            {link.type === 'prd' ? 'PRD' : link.type === 'tech' ? '技术方案' : '设计稿'}
                          </Badge>
                        )}
                      </a>
                    ))}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          {/* CodeGen History */}
          <Card>
            <CardHeader><CardTitle className="text-lg">生成历史</CardTitle></CardHeader>
            <CardContent>
              {req.codegen_tasks && req.codegen_tasks.length > 0 ? (
                <div className="space-y-3">
                  {req.codegen_tasks.map((task: CodeGenTask) => (
                    <div key={task.id}
                      className="flex items-center justify-between p-3 rounded-md border cursor-pointer hover:bg-muted/50"
                      onClick={() => navigate(`/codegen/${task.id}`)}>
                      <div className="flex items-center gap-3">
                        {taskStatusIcon(task.status)}
                        <div>
                          <p className="text-sm font-medium flex items-center gap-1.5">
                            任务 #{task.id}
                            {task.prompt === '手动提交' && (
                              <Badge variant="outline" className="text-[10px] px-1.5 py-0">手动提交</Badge>
                            )}
                          </p>
                          <p className="text-xs text-muted-foreground">
                            {task.target_branch} · {new Date(task.created_at).toLocaleString('zh-CN')}
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        {task.diff_stat && (
                          <span className="text-xs text-muted-foreground">
                            {task.diff_stat.files_changed} 文件
                            <span className="text-green-500 ml-1">+{task.diff_stat.additions}</span>
                            <span className="text-red-500 ml-1">-{task.diff_stat.deletions}</span>
                          </span>
                        )}
                        {task.error_message && (
                          <span className="text-xs text-destructive truncate max-w-[200px]">{task.error_message}</span>
                        )}
                        <ArrowRight className="w-4 h-4 text-muted-foreground" />
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground text-center py-4">暂无生成记录</p>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Sidebar Info */}
        <div className="space-y-6">
          <Card>
            <CardHeader><CardTitle className="text-lg">详细信息</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">创建者</span>
                <span>{req.creator.name}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">指派人</span>
                <span>{req.assignee?.name || '未指派'}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">目标仓库</span>
                <span>{req.repository?.name || '未关联'}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">创建时间</span>
                <span>{new Date(req.created_at).toLocaleDateString('zh-CN')}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">更新时间</span>
                <span>{new Date(req.updated_at).toLocaleDateString('zh-CN')}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">截止时间</span>
                {req.deadline ? (
                  <span className={`inline-flex items-center gap-1 ${
                    new Date(req.deadline) < new Date() && req.status !== 'merged' && req.status !== 'approved'
                      ? 'text-destructive font-medium' : ''
                  }`}>
                    {new Date(req.deadline).toLocaleDateString('zh-CN')}
                    {new Date(req.deadline) < new Date() && req.status !== 'merged' && req.status !== 'approved' && (
                      <AlertTriangle className="w-3.5 h-3.5" />
                    )}
                  </span>
                ) : (
                  <span className="text-muted-foreground">未设置</span>
                )}
              </div>
            </CardContent>
          </Card>

          {req.repository && (
            <Card>
              <CardHeader><CardTitle className="text-lg flex items-center gap-2"><GitBranch className="w-4 h-4" />分支信息</CardTitle></CardHeader>
              <CardContent className="space-y-3">
                <div className="space-y-1">
                  <span className="text-xs text-muted-foreground">分支名</span>
                  <div className="flex items-center gap-2">
                    <code className="text-sm bg-muted px-2 py-1 rounded flex-1 truncate">code-master/req-{req.id}</code>
                    <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0"
                      onClick={() => copyToClipboard(`code-master/req-${req.id}`)}>
                      <Copy className="w-3.5 h-3.5" />
                    </Button>
                  </div>
                </div>
                <div className="space-y-1.5">
                  <span className="text-xs text-muted-foreground">拉取远程分支（已存在）</span>
                  <div className="flex items-center gap-2">
                    <code className="text-xs bg-muted px-2 py-1 rounded flex-1 break-all leading-relaxed">
                      git fetch origin && git checkout -b code-master/req-{req.id} origin/code-master/req-{req.id}
                    </code>
                    <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0"
                      onClick={() => copyToClipboard(`git fetch origin && git checkout -b code-master/req-${req.id} origin/code-master/req-${req.id}`)}>
                      <Copy className="w-3.5 h-3.5" />
                    </Button>
                  </div>
                </div>
                <div className="space-y-1.5">
                  <span className="text-xs text-muted-foreground">新建本地分支（不存在）</span>
                  <div className="flex items-center gap-2">
                    <code className="text-xs bg-muted px-2 py-1 rounded flex-1 break-all leading-relaxed">
                      git checkout -b code-master/req-{req.id}
                    </code>
                    <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0"
                      onClick={() => copyToClipboard(`git checkout -b code-master/req-${req.id}`)}>
                      <Copy className="w-3.5 h-3.5" />
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {req.latest_review && (
            <Card>
              <CardHeader><CardTitle className="text-lg">最新审查</CardTitle></CardHeader>
              <CardContent className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">AI 评分</span>
                  <span className="font-medium">{req.latest_review.ai_score}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">人工审查</span>
                  <Badge variant={
                    req.latest_review.human_status === 'approved' ? 'success' :
                    req.latest_review.human_status === 'rejected' ? 'destructive' : 'outline'
                  } className="text-xs">
                    {req.latest_review.human_status === 'pending' ? '待审查' :
                     req.latest_review.human_status === 'approved' ? '已通过' : '已拒绝'}
                  </Badge>
                </div>
                <Button variant="outline" size="sm" className="w-full mt-2"
                  onClick={() => navigate(`/reviews/${req.latest_review!.id}`)}>
                  查看审查详情
                </Button>
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      {/* Edit Dialog */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className="max-w-xl max-h-[85vh] flex flex-col">
          <DialogHeader>
            <DialogTitle>编辑需求</DialogTitle>
            <DialogDescription>修改需求信息</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4 overflow-y-auto flex-1 min-h-0">
            <div className="space-y-2">
              <label className="text-sm font-medium">标题</label>
              <Input value={editData.title} onChange={(e) => setEditData({ ...editData, title: e.target.value })} />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">描述</label>
              <Textarea value={editData.description} onChange={(e) => setEditData({ ...editData, description: e.target.value })} rows={6} />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">优先级</label>
              <Select value={editData.priority} onValueChange={(v) => setEditData({ ...editData, priority: v })}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="p0">P0 - 紧急</SelectItem>
                  <SelectItem value="p1">P1 - 高</SelectItem>
                  <SelectItem value="p2">P2 - 中</SelectItem>
                  <SelectItem value="p3">P3 - 低</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">指派人</label>
                <Select value={editData.assignee_id} onValueChange={(v) => setEditData({ ...editData, assignee_id: v })}>
                  <SelectTrigger><SelectValue placeholder="选择指派人" /></SelectTrigger>
                  <SelectContent>
                    {members.map((m) => (
                      <SelectItem key={m.id} value={String(m.id)}>{m.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">目标仓库</label>
                <Select value={editData.repository_id} onValueChange={(v) => setEditData({ ...editData, repository_id: v })}>
                  <SelectTrigger><SelectValue placeholder="选择仓库" /></SelectTrigger>
                  <SelectContent>
                    {repos.map((r) => (
                      <SelectItem key={r.id} value={String(r.id)}>{r.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">期望完成时间</label>
              <Input type="date" value={editData.deadline}
                onClick={(e) => (e.target as HTMLInputElement).showPicker?.()}
                onChange={(e) => setEditData({ ...editData, deadline: e.target.value })} />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">关联文档</label>
              {editDocLinks.map((link, i) => (
                <div key={i} className="flex items-center gap-2">
                  <div className="flex-1 space-y-1">
                    <Input placeholder="粘贴飞书文档链接" value={link.url}
                      onChange={(e) => {
                        const updated = [...editDocLinks];
                        updated[i] = { ...updated[i], url: e.target.value };
                        setEditDocLinks(updated);
                      }}
                      onBlur={async () => {
                        const url = link.url.trim();
                        if (!url || link.title) return;
                        setEditDocResolving((prev) => ({ ...prev, [i]: true }));
                        setEditDocErrors((prev) => { const next = { ...prev }; delete next[i]; return next; });
                        try {
                          const res = await feishuApi.resolveDoc(url);
                          setEditDocLinks((prev) => {
                            const updated = [...prev];
                            if (updated[i]) updated[i] = { ...updated[i], title: res.title };
                            return updated;
                          });
                        } catch (err) {
                          const msg = (err as Error).message || '';
                          if (msg.includes('forBidden') || msg.includes('forbidden') || msg.includes('权限')) {
                            setEditDocErrors((prev) => ({ ...prev, [i]: '无权访问该文档，请在飞书中将文档设为「组织内链接可阅读」' }));
                          } else {
                            setEditDocErrors((prev) => ({ ...prev, [i]: '文档解析失败，保存后将使用链接作为标题' }));
                          }
                        }
                        finally { setEditDocResolving((prev) => ({ ...prev, [i]: false })); }
                      }} />
                    {editDocResolving[i] ? (
                      <div className="flex items-center gap-1 text-xs text-muted-foreground">
                        <Loader2 className="w-3 h-3 animate-spin" />解析中...
                      </div>
                    ) : editDocErrors[i] ? (
                      <p className="text-xs text-destructive">{editDocErrors[i]}</p>
                    ) : link.title ? (
                      <p className="text-xs text-muted-foreground truncate">{link.title}</p>
                    ) : null}
                  </div>
                  <Button variant="ghost" size="icon" className="shrink-0 h-9 w-9 text-muted-foreground hover:text-destructive"
                    onClick={() => {
                      setEditDocLinks(editDocLinks.filter((_, idx) => idx !== i));
                      setEditDocResolving((prev) => { const next = { ...prev }; delete next[i]; return next; });
                      setEditDocErrors((prev) => { const next = { ...prev }; delete next[i]; return next; });
                    }}>
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              ))}
              <Button variant="outline" size="sm" className="w-full"
                onClick={() => setEditDocLinks([...editDocLinks, { title: '', url: '' }])}>
                <Plus className="w-4 h-4 mr-1" />添加文档
              </Button>
              <p className="text-xs text-muted-foreground">粘贴飞书文档链接后自动解析标题</p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditOpen(false)}>取消</Button>
            <Button onClick={handleEdit} disabled={editLoading}>{editLoading ? '保存中...' : '保存'}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Generate Dialog */}
      <Dialog open={generateOpen} onOpenChange={setGenerateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>生成代码</DialogTitle>
            <DialogDescription>AI 将根据需求描述自动生成代码</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">补充说明（可选）</label>
              <Textarea placeholder="给 AI 的额外上下文信息..." value={extraContext}
                onChange={(e) => setExtraContext(e.target.value)} rows={3} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setGenerateOpen(false)}>取消</Button>
            <Button onClick={handleGenerate} disabled={generateLoading}>
              {generateLoading ? <><Spinner size="sm" className="mr-2" />启动中...</> : <><Play className="w-4 h-4 mr-1" />开始生成</>}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Dialog */}
      <DeleteConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="删除需求"
        description="确定要删除这个需求吗？此操作不可撤销。"
        onConfirm={async () => {
          await requirementApi.delete(req.id, true);
          toast({ title: '需求已删除', variant: 'success' });
          navigate(-1);
        }}
      />

      {/* Manual Submit Dialog */}
      <Dialog open={manualSubmitOpen} onOpenChange={setManualSubmitOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>手动提交代码</DialogTitle>
            <DialogDescription>
              请先将代码推送到 <code className="bg-muted px-1.5 py-0.5 rounded text-xs">code-master/req-{req.id}</code> 分支，然后粘贴 Commit 链接
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Commit 链接 <span className="text-destructive">*</span></label>
              <Input placeholder="粘贴 Git 仓库的 Commit URL..." value={manualCommitUrl}
                onChange={(e) => setManualCommitUrl(e.target.value)} />
              <p className="text-xs text-muted-foreground">例如：https://gitlab.com/group/repo/-/commit/abc123</p>
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">提交说明（可选）</label>
              <Textarea placeholder="描述你的代码变更..." value={manualCommitMessage}
                onChange={(e) => setManualCommitMessage(e.target.value)} rows={3} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setManualSubmitOpen(false)}>取消</Button>
            <Button onClick={handleManualSubmit} disabled={manualSubmitLoading || !manualCommitUrl.trim()}>
              {manualSubmitLoading ? <><Spinner size="sm" className="mr-2" />提交中...</> : <><Upload className="w-4 h-4 mr-1" />确认提交</>}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Review Trigger Dialog */}
      <Dialog open={reviewDialogOpen} onOpenChange={setReviewDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Users className="w-5 h-5" />发起 Review
            </DialogTitle>
            <DialogDescription>
              将对最新完成的生成任务 #{latestCompletedTask?.id} 发起 AI Review
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3 py-2">
            <p className="text-sm text-muted-foreground">选择 Reviewer（可多选）</p>
            {members.length === 0 ? (
              <p className="text-sm text-muted-foreground text-center py-4">暂无项目成员</p>
            ) : (
              <div className="grid grid-cols-3 gap-2 max-h-80 overflow-y-auto">
                {members.map((m) => {
                  const selected = selectedReviewers.includes(m.id);
                  return (
                    <label key={m.id}
                      className={`aspect-square flex flex-col items-center justify-center gap-2 p-2 rounded-lg cursor-pointer border-2 transition-all ${
                        selected
                          ? 'border-primary bg-primary/5 shadow-sm'
                          : 'border-muted hover:border-muted-foreground/30 hover:bg-muted/50'
                      }`}>
                      <input type="checkbox" className="sr-only" checked={selected}
                        onChange={() => toggleReviewer(m.id)} />
                      <div className="relative">
                        {m.avatar ? (
                          <img src={m.avatar} alt={m.name}
                            className={`w-12 h-12 rounded-full ring-2 ${selected ? 'ring-primary' : 'ring-border'}`} />
                        ) : (
                          <div className={`w-12 h-12 rounded-full flex items-center justify-center text-lg font-semibold ring-2 ${
                            selected ? 'bg-primary/20 text-primary ring-primary' : 'bg-muted text-muted-foreground ring-border'
                          }`}>
                            {m.name[0]}
                          </div>
                        )}
                        {selected && (
                          <div className="absolute -top-1 -right-1 w-5 h-5 rounded-full bg-primary flex items-center justify-center">
                            <CheckCircle className="w-3.5 h-3.5 text-primary-foreground" />
                          </div>
                        )}
                      </div>
                      <div className="text-center min-w-0 w-full">
                        <p className="text-sm font-medium truncate">{m.name}</p>
                        <Badge variant="outline" className="text-[10px] mt-0.5">{m.role}</Badge>
                      </div>
                    </label>
                  );
                })}
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setReviewDialogOpen(false)}>取消</Button>
            <Button onClick={handleTriggerReview} disabled={triggeringReview}>
              {triggeringReview ? <Spinner size="sm" className="mr-1" /> : null}
              发起 Review
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

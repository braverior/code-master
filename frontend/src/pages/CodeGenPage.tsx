import { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { codegenApi } from '@/api/codegen';
import { requirementApi } from '@/api/requirement';
import { projectApi } from '@/api/project';
import { useCodegenStream } from '@/hooks/use-codegen-stream';
import { useToast } from '@/hooks/use-toast';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Loading, Spinner } from '@/components/ui/spinner';
import { Progress } from '@/components/ui/progress';
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog';
import {
  Play, Square, ArrowLeft, FileCode, Eye, ChevronDown, ChevronRight,
  Terminal, FileEdit, FilePlus, BookOpen, CheckCircle, XCircle, Clock, Loader2,
  Info, AlertTriangle, AlertCircle, GitBranch, Cpu, Upload, ExternalLink,
  Search, FolderSearch, Wrench, ListTodo, ClipboardCheck, Users,
} from 'lucide-react';
import type { CodeGenTask, DiffFile, SSEOutputEvent, SSELogEvent, ProjectMember } from '@/types';

const stageOrder = ['pending', 'cloning', 'running', 'completed'];
const stageLabels: Record<string, string> = {
  pending: '等待中', cloning: '克隆仓库', running: '生成代码', completed: '已完成',
  failed: '失败', cancelled: '已取消',
};

function buildCommitUrl(gitUrl: string, platform: string, commitSha: string): string {
  const base = gitUrl.replace(/\.git$/, '');
  if (platform === 'github') {
    return `${base}/commit/${commitSha}`;
  }
  // GitLab
  return `${base}/-/commit/${commitSha}`;
}

export function CodeGenPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { toast } = useToast();
  const [task, setTask] = useState<CodeGenTask | null>(null);
  const [loading, setLoading] = useState(true);
  const [diffFiles, setDiffFiles] = useState<DiffFile[]>([]);
  const [showDiff, setShowDiff] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [reviewDialogOpen, setReviewDialogOpen] = useState(false);
  const [members, setMembers] = useState<ProjectMember[]>([]);
  const [selectedReviewers, setSelectedReviewers] = useState<number[]>([]);
  const [triggeringReview, setTriggeringReview] = useState(false);
  const outputEndRef = useRef<HTMLDivElement>(null);

  const taskId = id ? Number(id) : null;
  const isRunning = task?.status === 'pending' || task?.status === 'cloning' || task?.status === 'running';
  const isManualSubmit = task?.prompt === '手动提交';

  // Always enable stream: for running tasks it's real-time; for completed/failed tasks it replays history from Redis
  const stream = useCodegenStream({ taskId });

  // Fetch task details
  useEffect(() => {
    if (!taskId) return;
    codegenApi.get(taskId)
      .then((data) => {
        setTask(data);
        setLoading(false);
      })
      .catch((err) => {
        toast({ title: '获取任务失败', description: (err as Error).message, variant: 'destructive' });
        setLoading(false);
      });
  }, [taskId]);

  // Update task status from stream
  useEffect(() => {
    if (stream.status) {
      setTask((prev) => prev ? { ...prev, status: stream.status!.status } : prev);
    }
  }, [stream.status]);

  // Refetch task on done
  useEffect(() => {
    if (stream.done && taskId) {
      codegenApi.get(taskId).then(setTask);
    }
  }, [stream.done, taskId]);

  // Auto-scroll output
  useEffect(() => {
    outputEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [stream.entries]);

  const handleCancel = async () => {
    if (!taskId) return;
    setCancelling(true);
    try {
      await codegenApi.cancel(taskId);
      toast({ title: '任务已取消', variant: 'success' });
      codegenApi.get(taskId).then(setTask);
    } catch (err) {
      toast({ title: '取消失败', description: (err as Error).message, variant: 'destructive' });
    } finally {
      setCancelling(false);
    }
  };

  const handleViewDiff = async () => {
    if (!taskId) return;
    try {
      const data = await codegenApi.getDiff(taskId);
      setDiffFiles(data.files);
      setShowDiff(true);
    } catch (err) {
      toast({ title: '获取 Diff 失败', description: (err as Error).message, variant: 'destructive' });
    }
  };

  const handleOpenReviewDialog = async () => {
    if (!task?.requirement?.id) return;
    try {
      const req = await requirementApi.get(task.requirement.id);
      if (req.project?.id) {
        const proj = await projectApi.get(req.project.id);
        setMembers(proj.members ?? []);
      }
    } catch {
      // Proceed with empty members
    }
    setSelectedReviewers([]);
    setReviewDialogOpen(true);
  };

  const handleTriggerReview = async () => {
    if (!taskId) return;
    setTriggeringReview(true);
    try {
      await codegenApi.triggerReview(taskId, { reviewer_ids: selectedReviewers });
      toast({ title: 'Review 已发起', variant: 'success' });
      setReviewDialogOpen(false);
      codegenApi.get(taskId).then(setTask);
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

  if (loading) return <Loading />;
  if (!task) return <div className="p-6 text-center text-muted-foreground">任务不存在</div>;

  const currentStage = stageOrder.indexOf(task.status);
  const progressPercent = task.status === 'completed' ? 100 :
    task.status === 'failed' || task.status === 'cancelled' ? 0 :
    Math.max(((currentStage + 1) / stageOrder.length) * 100, 10);

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <Button variant="ghost" size="sm" className="mb-2" onClick={() => navigate(-1)}>
            <ArrowLeft className="w-4 h-4 mr-1" />返回
          </Button>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <FileCode className="w-6 h-6 text-primary" />
            代码生成 #{task.id}
          </h1>
          {task.requirement && (
            <p className="text-muted-foreground mt-1 cursor-pointer hover:text-foreground"
              onClick={() => navigate(`/requirements/${task.requirement!.id}`)}>
              {task.requirement.title}
            </p>
          )}
        </div>
        <div className="flex items-center gap-2">
          {isManualSubmit && (
            <Badge variant="outline">手动提交</Badge>
          )}
          <Badge variant={
            task.status === 'completed' ? 'success' :
            task.status === 'failed' || task.status === 'cancelled' ? 'destructive' :
            'warning'
          }>
            {stageLabels[task.status] || task.status}
          </Badge>
          {isRunning && (
            <Button variant="outline" size="sm" onClick={handleCancel} disabled={cancelling}>
              {cancelling ? <Spinner size="sm" /> : <><Square className="w-4 h-4 mr-1" />取消</>}
            </Button>
          )}
        </div>
      </div>

      {/* Stage Progress */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center gap-2 mb-3">
            {stageOrder.map((stage, i) => {
              const isActive = stageOrder.indexOf(task.status) >= i;
              const isCurrent = task.status === stage;
              return (
                <div key={stage} className="flex items-center gap-2 flex-1">
                  <div className={`flex items-center gap-1.5 text-xs font-medium ${
                    isActive ? 'text-primary' : 'text-muted-foreground'
                  }`}>
                    {isCurrent && isRunning ? (
                      <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    ) : isActive ? (
                      <CheckCircle className="w-3.5 h-3.5" />
                    ) : (
                      <Clock className="w-3.5 h-3.5" />
                    )}
                    {stageLabels[stage]}
                  </div>
                  {i < stageOrder.length - 1 && (
                    <div className={`flex-1 h-0.5 ${isActive ? 'bg-primary' : 'bg-muted'}`} />
                  )}
                </div>
              );
            })}
          </div>
          <Progress value={progressPercent} className="h-2"
            indicatorClassName={task.status === 'failed' ? 'bg-destructive' : undefined} />
          {stream.progress && (
            <div className="flex items-center gap-4 mt-3 text-xs text-muted-foreground">
              <span>读取: {stream.progress.files_read}</span>
              <span>写入: {stream.progress.files_written}</span>
              <span>编辑: {stream.progress.files_edited}</span>
              <span>轮次: {stream.progress.turns_used}/{stream.progress.max_turns}</span>
              {stream.progress.current_action && (
                <span className="truncate">{stream.progress.current_action}</span>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Unified Output Stream */}
        <div className="lg:col-span-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg flex items-center gap-2">
                <Terminal className="w-5 h-5" />
                实时输出
                {stream.connected && <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="bg-muted/50 rounded-md p-4 max-h-[600px] overflow-y-auto font-mono text-sm space-y-2">
                {isManualSubmit && (
                  <div className="rounded-md border border-primary/30 bg-primary/5 text-primary p-3 space-y-2">
                    <div className="flex items-center gap-2">
                      <Upload className="w-4 h-4 shrink-0" />
                      <span className="text-sm font-semibold">手动提交的代码</span>
                    </div>
                    <p className="text-xs opacity-80">此任务由开发者手动推送代码后提交，未经过 AI 自动生成。</p>
                    {task.commit_sha && task.repository?.git_url && (
                      <a
                        href={buildCommitUrl(task.repository.git_url, task.repository.platform, task.commit_sha)}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1.5 text-xs text-primary hover:underline"
                      >
                        <ExternalLink className="w-3.5 h-3.5" />
                        查看 Commit: <span className="font-mono">{task.commit_sha.substring(0, 8)}</span>
                      </a>
                    )}
                    {task.extra_context && (
                      <div className="pt-1">
                        <span className="text-[10px] font-semibold opacity-60">提交说明</span>
                        <p className="text-xs opacity-80 mt-0.5">{task.extra_context}</p>
                      </div>
                    )}
                  </div>
                )}
                {stream.entries.length === 0 && !stream.connected && stream.done && !isManualSubmit && (
                  <p className="text-muted-foreground text-center py-4">
                    {task.status === 'completed' ? '任务已完成（无输出记录）' : '暂无输出'}
                  </p>
                )}
                {stream.entries.length === 0 && !stream.done && !stream.connected && !isRunning && !isManualSubmit && (
                  <p className="text-muted-foreground text-center py-4">正在加载历史记录...</p>
                )}
                {(() => {
                  const merged = mergeStreamEntries(stream.entries);
                  // Track last tool name so tool_result knows which tool it belongs to
                  let lastToolName = '';
                  return merged.map((entry, i) => {
                    if (entry.kind === 'log') {
                      return <LogBlock key={i} log={entry.data as SSELogEvent} />;
                    }
                    const out = entry.data as SSEOutputEvent;
                    if (out.type === 'tool_use') {
                      lastToolName = out.tool || '';
                    }
                    const prevTool = out.type === 'tool_result' ? lastToolName : undefined;
                    if (out.type === 'tool_result') {
                      lastToolName = ''; // reset after consuming
                    }
                    return <OutputBlock key={i} output={out} prevTool={prevTool} />;
                  });
                })()}
                {stream.error && (
                  <div className="p-2 rounded bg-destructive/10 text-destructive text-xs">
                    {stream.error}
                  </div>
                )}
                <div ref={outputEndRef} />
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Task Info */}
        <div className="space-y-6">
          <Card>
            <CardHeader><CardTitle className="text-lg">任务信息</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">仓库</span>
                <span>{task.repository?.name || '-'}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">源分支</span>
                <span className="font-mono text-xs">{task.source_branch}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">目标分支</span>
                <span className="font-mono text-xs">{task.target_branch}</span>
              </div>
              {task.commit_sha && (
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Commit</span>
                  {task.repository?.git_url ? (
                    <a
                      href={buildCommitUrl(task.repository.git_url, task.repository.platform, task.commit_sha)}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="font-mono text-xs text-primary hover:underline flex items-center gap-1"
                    >
                      {task.commit_sha.substring(0, 8)}
                      <ExternalLink className="w-3 h-3" />
                    </a>
                  ) : (
                    <span className="font-mono text-xs">{task.commit_sha.substring(0, 8)}</span>
                  )}
                </div>
              )}
              {task.extra_context && (
                <div className="space-y-1 text-sm">
                  <span className="text-muted-foreground">补充说明</span>
                  <p className="text-xs bg-muted/50 rounded p-2 whitespace-pre-wrap">{task.extra_context}</p>
                </div>
              )}
              {task.started_at && (
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">开始时间</span>
                  <span>{new Date(task.started_at).toLocaleString('zh-CN')}</span>
                </div>
              )}
              {task.completed_at && (
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">完成时间</span>
                  <span>{new Date(task.completed_at).toLocaleString('zh-CN')}</span>
                </div>
              )}
              {task.claude_cost_usd != null && (
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">费用</span>
                  <span>${task.claude_cost_usd.toFixed(4)}</span>
                </div>
              )}
              {stream.status?.pid && (
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">进程 PID</span>
                  <span className="font-mono text-xs">{stream.status.pid}</span>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Diff Stats */}
          {task.diff_stat && task.status === 'completed' && (
            <Card>
              <CardHeader><CardTitle className="text-lg">变更统计</CardTitle></CardHeader>
              <CardContent className="space-y-3">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">变更文件</span>
                  <span className="font-medium">{task.diff_stat.files_changed}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">新增行</span>
                  <span className="text-green-500 font-medium">+{task.diff_stat.additions}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">删除行</span>
                  <span className="text-red-500 font-medium">-{task.diff_stat.deletions}</span>
                </div>
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" className="flex-1" onClick={handleViewDiff}>
                    <Eye className="w-4 h-4 mr-1" />查看 Diff
                  </Button>
                  {task.repository?.git_url && task.commit_sha && (
                    <a
                      href={buildCommitUrl(task.repository.git_url, task.repository.platform, task.commit_sha)}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex-1"
                    >
                      <Button variant="outline" size="sm" className="w-full">
                        <ExternalLink className="w-4 h-4 mr-1" />在 Git 仓库中查看
                      </Button>
                    </a>
                  )}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Review */}
          {task.status === 'completed' && !task.review && (
            <Card>
              <CardHeader><CardTitle className="text-lg">Review</CardTitle></CardHeader>
              <CardContent>
                <Button className="w-full" onClick={handleOpenReviewDialog}>
                  <ClipboardCheck className="w-4 h-4 mr-1" />发起 Review
                </Button>
              </CardContent>
            </Card>
          )}
          {task.review && (
            <Card>
              <CardHeader><CardTitle className="text-lg">Review</CardTitle></CardHeader>
              <CardContent className="space-y-3">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">AI 评分</span>
                  <span className="font-medium">{task.review.ai_score}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">AI 状态</span>
                  <Badge variant={task.review.ai_status === 'passed' ? 'success' : 'destructive'} className="text-xs">
                    {task.review.ai_status === 'passed' ? '通过' : task.review.ai_status}
                  </Badge>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">人工审查</span>
                  <Badge variant={
                    task.review.human_status === 'approved' ? 'success' :
                    task.review.human_status === 'rejected' ? 'destructive' : 'outline'
                  } className="text-xs">
                    {task.review.human_status === 'pending' ? '待审查' :
                     task.review.human_status === 'approved' ? '已通过' : '已拒绝'}
                  </Badge>
                </div>
                <Button variant="outline" size="sm" className="w-full"
                  onClick={() => navigate(`/reviews/${task.review!.id}`)}>
                  查看 Review 详情
                </Button>
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      {/* Diff Panel */}
      {showDiff && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-lg">代码变更</CardTitle>
            <Button variant="ghost" size="sm" onClick={() => setShowDiff(false)}>收起</Button>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {diffFiles.map((file) => (
                <DiffFileBlock key={file.path} file={file} />
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Review Trigger Dialog */}
      <Dialog open={reviewDialogOpen} onOpenChange={setReviewDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Users className="w-5 h-5" />发起 Review
            </DialogTitle>
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

// ---- Merge consecutive thinking entries into a single block ----

type MergedEntry =
  | { kind: 'log'; data: SSELogEvent }
  | { kind: 'output'; data: SSEOutputEvent };

function mergeStreamEntries(entries: import('@/types').StreamEntry[]): MergedEntry[] {
  const result: MergedEntry[] = [];
  for (const entry of entries) {
    if (entry.kind === 'output' && entry.data.type === 'thinking') {
      const last = result[result.length - 1];
      if (last && last.kind === 'output' && last.data.type === 'thinking') {
        // Merge into previous thinking block
        last.data = { ...last.data, content: (last.data.content || '') + (entry.data.content || '') };
        continue;
      }
    }
    // Similarly merge consecutive text blocks
    if (entry.kind === 'output' && entry.data.type === 'text') {
      const last = result[result.length - 1];
      if (last && last.kind === 'output' && last.data.type === 'text') {
        last.data = { ...last.data, content: (last.data.content || '') + (entry.data.content || '') };
        continue;
      }
    }
    result.push({ ...entry });
  }
  return result;
}

// ---- Log block: system operation logs displayed inline in the output timeline ----

const phaseIcon: Record<string, React.ReactNode> = {
  clone: <GitBranch className="w-3.5 h-3.5" />,
  claude: <Cpu className="w-3.5 h-3.5" />,
  push: <Upload className="w-3.5 h-3.5" />,
};

const phaseLabel: Record<string, string> = {
  clone: 'Git', claude: 'Claude Code', push: 'Push',
};

function LogBlock({ log }: { log: SSELogEvent }) {
  const isError = log.level === 'error';
  const isWarn = log.level === 'warn';
  // Auto-expand error logs with detail so users can immediately see the failure reason
  const [expanded, setExpanded] = useState(isError && !!log.detail);

  const levelIcon = isError ? <AlertCircle className="w-3.5 h-3.5" /> :
    isWarn ? <AlertTriangle className="w-3.5 h-3.5" /> :
    <Info className="w-3.5 h-3.5" />;

  const colorClass = isError ? 'border-destructive/50 bg-destructive/5 text-destructive' :
    isWarn ? 'border-yellow-500/50 bg-yellow-500/5 text-yellow-400' :
    'border-primary/30 bg-primary/5 text-primary';

  const hasDetail = log.detail && Object.keys(log.detail).length > 0;

  return (
    <div className={`rounded-md border p-2.5 ${colorClass}`}>
      <div className="flex items-center gap-2">
        <span className="shrink-0">{phaseIcon[log.phase] || levelIcon}</span>
        <span className="text-xs font-semibold shrink-0">{phaseLabel[log.phase] || log.phase}</span>
        <span className="text-xs">{log.message}</span>
        {log.detail?.pid != null && (
          <Badge variant="outline" className="text-[10px] px-1.5 py-0 h-4 font-mono">
            PID {String(log.detail.pid)}
          </Badge>
        )}
        {hasDetail && (
          <button onClick={() => setExpanded(!expanded)}
            className="ml-auto shrink-0 text-xs opacity-60 hover:opacity-100 cursor-pointer">
            {expanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
          </button>
        )}
      </div>
      {expanded && hasDetail && (
        <div className="mt-2 space-y-2">
          <LogDetail detail={log.detail!} />
        </div>
      )}
    </div>
  );
}

/** Renders log detail with special formatting for stderr and last_stdout_lines */
function LogDetail({ detail }: { detail: Record<string, unknown> }) {
  const { stderr, last_stdout_lines, ...rest } = detail as Record<string, unknown>;

  return (
    <>
      {typeof stderr === 'string' && (
        <div>
          <div className="text-[10px] font-semibold opacity-60 mb-1">STDERR</div>
          <pre className="text-[11px] opacity-80 whitespace-pre-wrap break-all bg-black/20 rounded p-2 max-h-60 overflow-auto">
            {stderr}
          </pre>
        </div>
      )}
      {Array.isArray(last_stdout_lines) && last_stdout_lines.length > 0 && (
        <div>
          <div className="text-[10px] font-semibold opacity-60 mb-1">最后 {last_stdout_lines.length} 行 STDOUT</div>
          <pre className="text-[11px] opacity-80 whitespace-pre-wrap break-all bg-black/20 rounded p-2 max-h-40 overflow-auto">
            {last_stdout_lines.map((l: unknown) => String(l)).join('\n')}
          </pre>
        </div>
      )}
      {Object.keys(rest).length > 0 && (
        <pre className="text-[11px] opacity-70 whitespace-pre-wrap break-all">
          {JSON.stringify(rest, null, 2)}
        </pre>
      )}
    </>
  );
}

// ---- Output block: Claude Code stdout events ----

const toolIconMap: Record<string, React.ReactNode> = {
  Read: <BookOpen className="w-3.5 h-3.5" />,
  Write: <FilePlus className="w-3.5 h-3.5" />,
  Edit: <FileEdit className="w-3.5 h-3.5" />,
  Bash: <Terminal className="w-3.5 h-3.5" />,
  Glob: <FolderSearch className="w-3.5 h-3.5" />,
  Grep: <Search className="w-3.5 h-3.5" />,
  Task: <Wrench className="w-3.5 h-3.5" />,
  TodoWrite: <ListTodo className="w-3.5 h-3.5" />,
};

/** Extract a short human-readable summary of the tool input */
function toolInputSummary(tool: string, input: Record<string, unknown>): string {
  switch (tool) {
    case 'Read':
    case 'Write':
    case 'Edit':
      return String(input.file_path || '');
    case 'Bash':
      return String(input.command || '');
    case 'Glob':
      return String(input.pattern || '');
    case 'Grep':
      return String(input.pattern || '');
    case 'Task':
      return String(input.description || input.prompt || '').substring(0, 80);
    case 'TodoWrite':
      return '更新任务列表';
    default:
      return JSON.stringify(input).substring(0, 100);
  }
}

function OutputBlock({ output, prevTool }: { output: SSEOutputEvent; prevTool?: string }) {
  const isThinking = output.type === 'thinking';
  const isLongText = output.type === 'text' && (output.content || '').length > 500;
  const isLongResult = output.type === 'tool_result' && (output.output || '').split('\n').length > 6;
  const [collapsed, setCollapsed] = useState(isThinking || isLongText || isLongResult);

  if (output.type === 'thinking') {
    return (
      <div className="border-l-2 border-muted-foreground/30 pl-3">
        <button onClick={() => setCollapsed(!collapsed)}
          className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground cursor-pointer">
          {collapsed ? <ChevronRight className="w-3 h-3" /> : <ChevronDown className="w-3 h-3" />}
          思考中...
        </button>
        {!collapsed && output.content && (
          <pre className="text-xs text-muted-foreground mt-1 whitespace-pre-wrap max-h-60 overflow-auto">{output.content}</pre>
        )}
      </div>
    );
  }

  if (output.type === 'text') {
    const content = output.content || '';
    const isLong = content.length > 500;
    return (
      <div>
        <pre className={`whitespace-pre-wrap text-foreground text-sm ${isLong && collapsed ? 'max-h-32 overflow-hidden' : ''}`}>
          {content}
        </pre>
        {isLong && (
          <button onClick={() => setCollapsed(!collapsed)}
            className="text-xs text-primary hover:underline mt-1 cursor-pointer">
            {collapsed ? '展开全部' : '收起'}
          </button>
        )}
      </div>
    );
  }

  if (output.type === 'tool_use') {
    const input = (output.input || {}) as Record<string, unknown>;

    // TodoWrite: render as task checklist
    if (output.tool === 'TodoWrite') {
      const todos = (input.todos || []) as Array<Record<string, unknown>>;
      if (todos.length === 0) return null;
      return (
        <div className="rounded border border-muted p-2.5 space-y-1.5">
          <div className="flex items-center gap-1.5 text-xs font-semibold text-muted-foreground">
            <ListTodo className="w-3.5 h-3.5" />
            任务列表
          </div>
          {todos.map((todo, i) => {
            const status = String(todo.status || 'pending');
            return (
              <div key={i} className="flex items-start gap-2 text-xs">
                <span className="mt-0.5 shrink-0">
                  {status === 'completed' ? <CheckCircle className="w-3.5 h-3.5 text-green-500" /> :
                   status === 'in_progress' ? <Loader2 className="w-3.5 h-3.5 text-primary animate-spin" /> :
                   <Clock className="w-3.5 h-3.5 text-muted-foreground" />}
                </span>
                <span className={status === 'completed' ? 'line-through text-muted-foreground' : 'text-foreground'}>
                  {String(todo.content || todo.activeForm || '')}
                </span>
              </div>
            );
          })}
        </div>
      );
    }

    // Task (subagent): show description
    if (output.tool === 'Task') {
      const desc = String(input.description || '');
      const prompt = String(input.prompt || '');
      return (
        <div className="flex items-start gap-2 p-2 rounded bg-accent/50">
          <span className="text-primary mt-0.5 shrink-0"><Wrench className="w-3.5 h-3.5" /></span>
          <div className="flex-1 min-w-0">
            <span className="text-xs font-semibold text-primary">Task</span>
            {desc && <span className="text-xs text-muted-foreground ml-1.5">{desc}</span>}
            {prompt && (
              <p className="text-xs text-muted-foreground mt-0.5 truncate font-mono">{prompt.substring(0, 120)}</p>
            )}
          </div>
        </div>
      );
    }

    // Default tool_use
    const summary = toolInputSummary(output.tool || '', input);
    return (
      <div className="flex items-start gap-2 p-2 rounded bg-accent/50">
        <span className="text-primary mt-0.5 shrink-0">{toolIconMap[output.tool || ''] || <Play className="w-3.5 h-3.5" />}</span>
        <div className="flex-1 min-w-0">
          <span className="text-xs font-semibold text-primary">{output.tool}</span>
          {summary && (
            <p className="text-xs text-muted-foreground font-mono truncate mt-0.5">{summary}</p>
          )}
        </div>
      </div>
    );
  }

  if (output.type === 'tool_result') {
    // Hide verbose system messages from TodoWrite results
    if (prevTool === 'TodoWrite') return null;
    // Hide verbose system messages from Task results
    if (prevTool === 'Task') {
      const content = output.output || '';
      if (!content || content.length > 2000) return null; // Task agent returns very long output
      const lines = content.split('\n');
      const isLong = lines.length > 8;
      const preview = isLong && collapsed ? lines.slice(0, 6).join('\n') + '\n...' : content;
      return (
        <div className="ml-6 border-l border-muted pl-3">
          <pre className="text-[11px] text-muted-foreground whitespace-pre-wrap break-all max-h-48 overflow-auto">
            {preview}
          </pre>
          {isLong && (
            <button onClick={() => setCollapsed(!collapsed)}
              className="text-[11px] text-primary hover:underline cursor-pointer">
              {collapsed ? `展开全部 (${lines.length} 行)` : '收起'}
            </button>
          )}
        </div>
      );
    }

    const content = output.output || '';
    if (!content) return null;
    const lines = content.split('\n');
    const isLong = lines.length > 6;
    const preview = isLong && collapsed ? lines.slice(0, 5).join('\n') + '\n...' : content;

    return (
      <div className="ml-6 border-l border-muted pl-3">
        <pre className="text-[11px] text-muted-foreground whitespace-pre-wrap break-all max-h-48 overflow-auto">
          {preview}
        </pre>
        {isLong && (
          <button onClick={() => setCollapsed(!collapsed)}
            className="text-[11px] text-primary hover:underline cursor-pointer">
            {collapsed ? `展开全部 (${lines.length} 行)` : '收起'}
          </button>
        )}
      </div>
    );
  }

  if (output.type === 'result') {
    return null; // result 事件不需要显示，信息已在 done 事件和任务信息卡片中
  }

  return null;
}

// ---- Diff file block ----

function DiffFileBlock({ file }: { file: DiffFile }) {
  const [expanded, setExpanded] = useState(true);
  const statusIcon = file.status === 'added' ? <FilePlus className="w-3.5 h-3.5 text-green-500" /> :
    file.status === 'deleted' ? <XCircle className="w-3.5 h-3.5 text-red-500" /> :
    <FileEdit className="w-3.5 h-3.5 text-yellow-500" />;

  return (
    <div className="border rounded-md overflow-hidden">
      <button onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-2 p-2 bg-muted/50 hover:bg-muted text-sm cursor-pointer">
        {expanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
        {statusIcon}
        <span className="font-mono text-xs">{file.path}</span>
        <span className="ml-auto text-xs text-muted-foreground">
          <span className="text-green-500">+{file.additions}</span>
          <span className="text-red-500 ml-2">-{file.deletions}</span>
        </span>
      </button>
      {expanded && file.diff && (
        <pre className="p-3 text-xs overflow-x-auto bg-background">
          {file.diff.split('\n').map((line, i) => (
            <div key={i} className={
              line.startsWith('+') ? 'bg-green-500/10 text-green-400' :
              line.startsWith('-') ? 'bg-red-500/10 text-red-400' :
              line.startsWith('@@') ? 'text-blue-400' : 'text-muted-foreground'
            }>
              {line}
            </div>
          ))}
        </pre>
      )}
    </div>
  );
}

import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { reviewApi } from '@/api/review';
import { useToast } from '@/hooks/use-toast';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Textarea } from '@/components/ui/textarea';
import { Loading, Spinner } from '@/components/ui/spinner';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import {
  ArrowLeft, ClipboardCheck, AlertTriangle, Info, CheckCircle, XCircle,
  ExternalLink, GitMerge, Users,
} from 'lucide-react';
import type { Review } from '@/types';

function buildCompareUrl(gitUrl: string, platform: string, sourceBranch: string, targetBranch: string): string {
  const base = gitUrl.replace(/\.git$/, '');
  if (platform === 'github') {
    return `${base}/compare/${sourceBranch}...${targetBranch}`;
  }
  return `${base}/-/compare/${sourceBranch}...${targetBranch}`;
}

export function ReviewPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { toast } = useToast();
  const [review, setReview] = useState<Review | null>(null);
  const [loading, setLoading] = useState(true);
  const [reviewComment, setReviewComment] = useState('');
  const [reviewStatus, setReviewStatus] = useState('approved');
  const [submitting, setSubmitting] = useState(false);
  const [creatingMR, setCreatingMR] = useState(false);

  const fetchReview = async () => {
    if (!id) return;
    try {
      const data = await reviewApi.get(Number(id));
      setReview(data);
    } catch (err) {
      toast({ title: '获取 Review 失败', description: (err as Error).message, variant: 'destructive' });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchReview(); }, [id]);

  const handleSubmitReview = async () => {
    if (!review) return;
    if ((reviewStatus === 'rejected' || reviewStatus === 'needs_revision') && !reviewComment.trim()) {
      toast({ title: '拒绝时必须填写审查意见', variant: 'destructive' });
      return;
    }
    setSubmitting(true);
    try {
      await reviewApi.submitHumanReview(review.id, { comment: reviewComment, status: reviewStatus });
      toast({ title: '审查已提交', variant: 'success' });
      fetchReview();
    } catch (err) {
      toast({ title: '提交失败', description: (err as Error).message, variant: 'destructive' });
    } finally {
      setSubmitting(false);
    }
  };

  const handleCreateMR = async () => {
    if (!review) return;
    setCreatingMR(true);
    try {
      const result = await reviewApi.createMergeRequest(review.id);
      toast({ title: '合并请求已创建', variant: 'success' });
      if (result.merge_request_url) {
        window.open(result.merge_request_url, '_blank');
      }
      fetchReview();
    } catch (err) {
      toast({ title: '创建失败', description: (err as Error).message, variant: 'destructive' });
    } finally {
      setCreatingMR(false);
    }
  };

  if (loading) return <Loading />;
  if (!review) return <div className="p-6 text-center text-muted-foreground">Review 不存在</div>;

  const severityIcon = (s: string) => {
    switch (s) {
      case 'error': return <XCircle className="w-4 h-4 text-red-500" />;
      case 'warning': return <AlertTriangle className="w-4 h-4 text-yellow-500" />;
      case 'info': return <Info className="w-4 h-4 text-blue-500" />;
      default: return null;
    }
  };

  const categoryIcon = (status: string) => {
    switch (status) {
      case 'passed': return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'warning': return <AlertTriangle className="w-4 h-4 text-yellow-500" />;
      case 'failed': return <XCircle className="w-4 h-4 text-red-500" />;
      default: return null;
    }
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
            <ClipboardCheck className="w-6 h-6 text-primary" />
            代码审查 #{review.id}
          </h1>
          {review.requirement && (
            <p className="text-muted-foreground mt-1">{review.requirement.title}</p>
          )}
        </div>
        <div className="flex items-center gap-2">
          {review.human_status === 'approved' && review.merge_status === 'none' && (
            <Button onClick={handleCreateMR} disabled={creatingMR}>
              {creatingMR ? <Spinner size="sm" className="mr-2" /> : <GitMerge className="w-4 h-4 mr-1" />}
              创建合并请求
            </Button>
          )}
          {review.merge_request_url && (
            <Button variant="outline" onClick={() => window.open(review.merge_request_url!, '_blank')}>
              <ExternalLink className="w-4 h-4 mr-1" />查看 MR
            </Button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          {/* AI Review */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg flex items-center justify-between">
                <span>AI 审查结果</span>
                {review.ai_status === 'running' ? (
                  <Badge variant="warning"><Spinner size="sm" className="mr-1" />审查中</Badge>
                ) : review.ai_score != null ? (
                  <Badge variant={review.ai_score >= 80 ? 'success' : review.ai_score >= 60 ? 'warning' : 'destructive'}>
                    评分: {review.ai_score}
                  </Badge>
                ) : null}
              </CardTitle>
            </CardHeader>
            <CardContent>
              {review.ai_status === 'running' ? (
                <div className="flex flex-col items-center py-8">
                  <Spinner size="lg" />
                  <p className="mt-3 text-sm text-muted-foreground">AI 正在审查代码...</p>
                </div>
              ) : review.ai_review ? (
                <div className="space-y-4">
                  <p className="text-sm">{review.ai_review.summary}</p>

                  {/* Categories */}
                  {review.ai_review.categories && (
                    <div className="grid grid-cols-2 gap-3">
                      {Object.entries(review.ai_review.categories).map(([key, cat]) => (
                        <div key={key} className="flex items-center gap-2 p-2 rounded-md bg-muted/50">
                          {categoryIcon(cat.status)}
                          <div>
                            <p className="text-xs font-medium capitalize">{key.replace(/_/g, ' ')}</p>
                            <p className="text-xs text-muted-foreground">{cat.details}</p>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}

                  {/* Issues */}
                  {review.ai_review.issues && review.ai_review.issues.length > 0 && (
                    <div>
                      <p className="text-sm font-medium mb-2">发现的问题</p>
                      <div className="space-y-2">
                        {review.ai_review.issues.map((issue, i) => (
                          <div key={i} className="p-3 rounded-md border">
                            <div className="flex items-center gap-2 mb-1">
                              {severityIcon(issue.severity)}
                              <span className="text-sm font-medium">{issue.message}</span>
                            </div>
                            <p className="text-xs text-muted-foreground font-mono">
                              {issue.file}:{issue.line}
                            </p>
                            {issue.code_snippet && (
                              <pre className="text-xs bg-muted/50 p-2 rounded mt-1 overflow-x-auto">
                                {issue.code_snippet}
                              </pre>
                            )}
                            {issue.suggestion && (
                              <p className="text-xs text-muted-foreground mt-1">
                                建议: {issue.suggestion}
                              </p>
                            )}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground text-center py-4">暂无 AI 审查结果</p>
              )}
            </CardContent>
          </Card>

          {/* Human Review Form */}
          {(review.human_status === 'pending' || review.human_status === 'needs_revision') && review.ai_status !== 'running' && (
            <Card>
              <CardHeader><CardTitle className="text-lg">人工审查</CardTitle></CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium">审查结果</label>
                  <Select value={reviewStatus} onValueChange={setReviewStatus}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="approved">通过</SelectItem>
                      <SelectItem value="needs_revision">需要修改</SelectItem>
                      <SelectItem value="rejected">拒绝</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium">
                    审查意见 {(reviewStatus === 'rejected' || reviewStatus === 'needs_revision') && '*'}
                  </label>
                  <Textarea placeholder="输入审查意见..." value={reviewComment}
                    onChange={(e) => setReviewComment(e.target.value)} rows={4} />
                </div>
                <Button onClick={handleSubmitReview} disabled={submitting}>
                  {submitting ? '提交中...' : '提交审查'}
                </Button>
              </CardContent>
            </Card>
          )}

          {/* Existing Human Review */}
          {review.human_status !== 'pending' && review.human_reviewer && (
            <Card>
              <CardHeader>
                <CardTitle className="text-lg flex items-center justify-between">
                  <span>人工审查结果</span>
                  <Badge variant={
                    review.human_status === 'approved' ? 'success' :
                    review.human_status === 'rejected' ? 'destructive' : 'warning'
                  }>
                    {review.human_status === 'approved' ? '通过' :
                     review.human_status === 'rejected' ? '拒绝' : '需要修改'}
                  </Badge>
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  <p className="text-sm text-muted-foreground">审查人: {review.human_reviewer.name}</p>
                  {review.human_comment && (
                    <p className="text-sm whitespace-pre-wrap">{review.human_comment}</p>
                  )}
                </div>
              </CardContent>
            </Card>
          )}
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          <Card>
            <CardHeader><CardTitle className="text-lg">审查信息</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              {review.requirement && (
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">需求</span>
                  <span className="cursor-pointer text-primary hover:underline truncate max-w-[180px]"
                    onClick={() => navigate(`/requirements/${review.requirement!.id}`)}>
                    {review.requirement.title}
                  </span>
                </div>
              )}
              {review.repository && (
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">仓库</span>
                  <span>{review.repository.name}</span>
                </div>
              )}
              {review.source_branch && (
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">源分支</span>
                  <span className="font-mono text-xs">{review.source_branch}</span>
                </div>
              )}
              {review.target_branch && (
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">目标分支</span>
                  <span className="font-mono text-xs">{review.target_branch}</span>
                </div>
              )}
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">生成任务</span>
                <span className="cursor-pointer text-primary hover:underline"
                  onClick={() => navigate(`/codegen/${review.codegen_task_id}`)}>
                  #{review.codegen_task_id}
                </span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">AI 状态</span>
                <Badge variant={review.ai_status === 'passed' ? 'success' : review.ai_status === 'failed' ? 'destructive' : 'outline'} className="text-xs">
                  {review.ai_status === 'passed' ? '通过' : review.ai_status === 'running' ? '进行中' : review.ai_status}
                </Badge>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">人工审查</span>
                <Badge variant={
                  review.human_status === 'approved' ? 'success' :
                  review.human_status === 'rejected' ? 'destructive' : 'outline'
                } className="text-xs">
                  {review.human_status === 'pending' ? '待审查' :
                   review.human_status === 'approved' ? '已通过' :
                   review.human_status === 'rejected' ? '已拒绝' : '需修改'}
                </Badge>
              </div>
              {review.merge_status !== 'none' && (
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">MR 状态</span>
                  <Badge variant={review.merge_status === 'merged' ? 'success' : 'outline'} className="text-xs">
                    {review.merge_status === 'created' ? '已创建' :
                     review.merge_status === 'merged' ? '已合并' : review.merge_status}
                  </Badge>
                </div>
              )}
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">创建时间</span>
                <span>{new Date(review.created_at).toLocaleString('zh-CN')}</span>
              </div>
            </CardContent>
          </Card>

          {/* Reviewers */}
          {review.reviewers && review.reviewers.length > 0 && (
            <Card>
              <CardHeader><CardTitle className="text-lg flex items-center gap-2"><Users className="w-4 h-4" />Reviewers</CardTitle></CardHeader>
              <CardContent className="space-y-2">
                {review.reviewers.map((r) => (
                  <div key={r.id} className="flex items-center gap-2 text-sm">
                    {r.avatar ? (
                      <img src={r.avatar} alt={r.name} className="w-6 h-6 rounded-full" />
                    ) : (
                      <span className="w-6 h-6 rounded-full bg-primary/10 flex items-center justify-center text-xs font-medium text-primary">
                        {r.name.charAt(0)}
                      </span>
                    )}
                    <span>{r.name}</span>
                  </div>
                ))}
              </CardContent>
            </Card>
          )}

          {review.diff_stat && (
            <Card>
              <CardHeader><CardTitle className="text-lg">变更统计</CardTitle></CardHeader>
              <CardContent className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">变更文件</span>
                  <span>{review.diff_stat.files_changed}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">新增行</span>
                  <span className="text-green-500">+{review.diff_stat.additions}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">删除行</span>
                  <span className="text-red-500">-{review.diff_stat.deletions}</span>
                </div>
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" className="flex-1"
                    onClick={() => navigate(`/codegen/${review.codegen_task_id}`)}>
                    查看代码变更
                  </Button>
                  {review.git_url && review.source_branch && review.target_branch && (
                    <a
                      href={buildCompareUrl(review.git_url, review.platform || 'gitlab', review.source_branch, review.target_branch)}
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
        </div>
      </div>
    </div>
  );
}

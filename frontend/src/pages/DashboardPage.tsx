import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { dashboardApi } from '@/api/dashboard';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Loading } from '@/components/ui/spinner';
import { FolderKanban, FileText, ClipboardCheck, Zap, Activity, ArrowRight, BookOpen } from 'lucide-react';
import { Button } from '@/components/ui/button';
import type { DashboardStats, MyTasks } from '@/types';

export function DashboardPage() {
  const navigate = useNavigate();
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [myTasks, setMyTasks] = useState<MyTasks | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      dashboardApi.getStats().catch(() => null),
      dashboardApi.getMyTasks().catch(() => null),
    ]).then(([s, t]) => {
      setStats(s);
      setMyTasks(t);
      setLoading(false);
    });
  }, []);

  if (loading) return <Loading />;

  const statCards = [
    { label: '我的项目', value: stats?.my_projects ?? 0, icon: FolderKanban, color: 'text-blue-500' },
    { label: '待处理需求', value: stats?.my_open_requirements ?? 0, icon: FileText, color: 'text-orange-500' },
    { label: '待审查', value: stats?.my_pending_reviews ?? 0, icon: ClipboardCheck, color: 'text-purple-500' },
    { label: '运行中任务', value: stats?.codegen_running ?? 0, icon: Zap, color: 'text-green-500' },
  ];

  const activityTypeLabel = (type: string) => {
    switch (type) {
      case 'codegen_completed': return '代码生成完成';
      case 'codegen_failed': return '代码生成失败';
      case 'review_approved': return '审查通过';
      case 'review_rejected': return '审查拒绝';
      case 'requirement_created': return '需求创建';
      case 'mr_merged': return 'MR 已合并';
      default: return type;
    }
  };

  const activityVariant = (type: string) => {
    if (type.includes('completed') || type.includes('approved') || type.includes('merged')) return 'success' as const;
    if (type.includes('failed') || type.includes('rejected')) return 'destructive' as const;
    return 'secondary' as const;
  };

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <Activity className="w-6 h-6 text-primary" />
          仪表盘
        </h1>
        <p className="text-muted-foreground mt-1">项目概览和待办事项</p>
      </div>

      {/* Stat Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {statCards.map((item) => {
          const Icon = item.icon;
          return (
            <Card key={item.label}>
              <CardContent className="p-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-muted-foreground">{item.label}</p>
                    <p className="text-3xl font-bold mt-1">{item.value}</p>
                  </div>
                  <div className={`w-12 h-12 rounded-lg bg-muted flex items-center justify-center ${item.color}`}>
                    <Icon className="w-6 h-6" />
                  </div>
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>

      {/* Guide Entry */}
      <Card className="border-dashed cursor-pointer hover:border-primary/50 transition-colors" onClick={() => navigate('/guide')}>
        <CardContent className="p-4">
          <div className="flex items-center gap-4">
            <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
              <BookOpen className="w-5 h-5" />
            </div>
            <div className="flex-1 min-w-0">
              <h3 className="font-medium text-sm">新手入门</h3>
              <p className="text-xs text-muted-foreground mt-0.5">了解 CodeMaster 的核心工作流程：从创建项目到代码合并的完整流程</p>
            </div>
            <Button variant="outline" size="sm" onClick={(e) => { e.stopPropagation(); navigate('/guide'); }}>
              查看使用指南 <ArrowRight className="w-3.5 h-3.5 ml-1" />
            </Button>
          </div>
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Recent Activity */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">最近活动</CardTitle>
          </CardHeader>
          <CardContent>
            {stats?.recent_activity && stats.recent_activity.length > 0 ? (
              <div className="space-y-3">
                {stats.recent_activity.map((activity, i) => (
                  <div key={i} className="flex items-start gap-3 p-2 rounded-md hover:bg-muted/50">
                    <Badge variant={activityVariant(activity.type)} className="mt-0.5 shrink-0">
                      {activityTypeLabel(activity.type)}
                    </Badge>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm truncate">{activity.requirement?.title}</p>
                      <p className="text-xs text-muted-foreground">
                        {activity.project?.name} · {new Date(activity.time).toLocaleString('zh-CN')}
                      </p>
                    </div>
                    {activity.requirement && (
                      <Button size="sm" variant="ghost" className="shrink-0 text-xs"
                        onClick={() => navigate(`/requirements/${activity.requirement!.id}`)}>
                        查看 <ArrowRight className="w-3 h-3 ml-1" />
                      </Button>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground text-center py-8">暂无活动</p>
            )}
          </CardContent>
        </Card>

        {/* My Tasks */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">我的待办</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Running Tasks */}
            {myTasks?.running_tasks && myTasks.running_tasks.length > 0 && (
              <div>
                <p className="text-xs font-medium text-muted-foreground mb-2">运行中的任务</p>
                {myTasks.running_tasks.map((task) => (
                  <div key={task.task_id} className="flex items-center justify-between p-2 rounded-md hover:bg-muted/50">
                    <div className="flex-1 min-w-0">
                      <p className="text-sm truncate">{task.requirement.title}</p>
                      <p className="text-xs text-muted-foreground">
                        {new Date(task.started_at).toLocaleString('zh-CN')}
                      </p>
                    </div>
                    <Button size="sm" variant="outline" onClick={() => navigate(`/codegen/${task.task_id}`)}>
                      查看 <ArrowRight className="w-3 h-3 ml-1" />
                    </Button>
                  </div>
                ))}
              </div>
            )}

            {/* Pending Reviews */}
            {myTasks?.pending_reviews && myTasks.pending_reviews.length > 0 && (
              <div>
                <p className="text-xs font-medium text-muted-foreground mb-2">待审查</p>
                {myTasks.pending_reviews.map((review) => (
                  <div key={review.review_id} className="flex items-center justify-between p-2 rounded-md hover:bg-muted/50">
                    <div className="flex-1 min-w-0">
                      <p className="text-sm truncate">{review.requirement.title}</p>
                      <p className="text-xs text-muted-foreground">
                        AI 评分: {review.ai_score}
                      </p>
                    </div>
                    <Button size="sm" variant="outline" onClick={() => navigate(`/reviews/${review.review_id}`)}>
                      审查 <ArrowRight className="w-3 h-3 ml-1" />
                    </Button>
                  </div>
                ))}
              </div>
            )}

            {/* Pending Generate */}
            {myTasks?.pending_generate && myTasks.pending_generate.length > 0 && (
              <div>
                <p className="text-xs font-medium text-muted-foreground mb-2">待生成</p>
                {myTasks.pending_generate.map((req) => (
                  <div key={req.requirement_id} className="flex items-center justify-between p-2 rounded-md hover:bg-muted/50">
                    <div className="flex-1 min-w-0">
                      <p className="text-sm truncate">{req.title}</p>
                      <p className="text-xs text-muted-foreground">
                        {req.project.name} · {req.priority.toUpperCase()}
                      </p>
                    </div>
                    <Button size="sm" variant="outline" onClick={() => navigate(`/requirements/${req.requirement_id}`)}>
                      查看 <ArrowRight className="w-3 h-3 ml-1" />
                    </Button>
                  </div>
                ))}
              </div>
            )}

            {!myTasks?.running_tasks?.length && !myTasks?.pending_reviews?.length && !myTasks?.pending_generate?.length && (
              <p className="text-sm text-muted-foreground text-center py-8">暂无待办事项</p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

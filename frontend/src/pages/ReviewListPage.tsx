import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { reviewApi } from '@/api/review';
import { useToast } from '@/hooks/use-toast';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Loading } from '@/components/ui/spinner';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { ClipboardCheck, ArrowRight } from 'lucide-react';
import type { Review } from '@/types';

type TabKey = 'pending' | 'approved' | 'rejected' | 'all';

const TAB_CONFIG: { key: TabKey; label: string; humanStatus: string }[] = [
  { key: 'pending', label: '待审核', humanStatus: 'pending' },
  { key: 'approved', label: '已通过', humanStatus: 'approved' },
  { key: 'rejected', label: '已拒绝', humanStatus: 'rejected' },
  { key: 'all', label: '全部', humanStatus: '' },
];

function humanStatusLabel(status: string) {
  switch (status) {
    case 'pending': return '待审查';
    case 'needs_revision': return '需修改';
    case 'approved': return '已通过';
    case 'rejected': return '已拒绝';
    default: return status;
  }
}

function humanStatusVariant(status: string) {
  switch (status) {
    case 'approved': return 'success' as const;
    case 'rejected': return 'destructive' as const;
    case 'needs_revision': return 'warning' as const;
    default: return 'outline' as const;
  }
}

function ReviewTable({ reviews, navigate }: { reviews: Review[]; navigate: (path: string) => void }) {
  if (reviews.length === 0) {
    return <p className="text-sm text-muted-foreground text-center py-8">暂无审查记录</p>;
  }

  return (
    <table className="w-full">
      <thead>
        <tr className="border-b text-left text-sm text-muted-foreground">
          <th className="pb-2 font-medium">需求</th>
          <th className="pb-2 font-medium w-28">项目</th>
          <th className="pb-2 font-medium w-28">仓库</th>
          <th className="pb-2 font-medium w-20">AI 评价</th>
          <th className="pb-2 font-medium w-24">审查状态</th>
          <th className="pb-2 font-medium w-16">发起人</th>
          <th className="pb-2 font-medium w-16">Review</th>
          <th className="pb-2 font-medium w-28">变更</th>
          <th className="pb-2 font-medium w-28">时间</th>
          <th className="pb-2 font-medium w-10"></th>
        </tr>
      </thead>
      <tbody>
        {reviews.map((review) => (
          <tr key={review.id}
            className="border-b last:border-0 hover:bg-muted/50 cursor-pointer"
            onClick={() => navigate(`/reviews/${review.id}`)}>
            <td className="py-3">
              <div className="text-sm">{review.requirement?.title}</div>
              {review.ai_summary && (
                <div className="text-xs text-muted-foreground mt-0.5 line-clamp-1">{review.ai_summary}</div>
              )}
            </td>
            <td className="py-3 text-sm text-muted-foreground">{review.project?.name}</td>
            <td className="py-3 text-sm text-muted-foreground">{review.repository?.name}</td>
            <td className="py-3">
              {review.ai_score != null && (
                <Badge variant={review.ai_score >= 80 ? 'success' : review.ai_score >= 60 ? 'warning' : 'destructive'} className="text-xs">
                  {review.ai_score}
                </Badge>
              )}
            </td>
            <td className="py-3">
              <Badge variant={humanStatusVariant(review.human_status)} className="text-xs">
                {humanStatusLabel(review.human_status)}
              </Badge>
            </td>
            <td className="py-3">
              {review.creator ? (
                review.creator.avatar ? (
                  <img src={review.creator.avatar} alt={review.creator.name} title={review.creator.name} className="w-7 h-7 rounded-full" />
                ) : (
                  <div title={review.creator.name} className="w-7 h-7 rounded-full bg-primary/20 flex items-center justify-center text-xs font-medium">
                    {review.creator.name[0]}
                  </div>
                )
              ) : <span className="text-sm text-muted-foreground">-</span>}
            </td>
            <td className="py-3">
              {review.human_reviewer ? (
                review.human_reviewer.avatar ? (
                  <img src={review.human_reviewer.avatar} alt={review.human_reviewer.name} title={review.human_reviewer.name} className="w-7 h-7 rounded-full" />
                ) : review.human_reviewer.name ? (
                  <div title={review.human_reviewer.name} className="w-7 h-7 rounded-full bg-primary/20 flex items-center justify-center text-xs font-medium">
                    {review.human_reviewer.name[0]}
                  </div>
                ) : <span className="text-sm text-muted-foreground">-</span>
              ) : <span className="text-sm text-muted-foreground">-</span>}
            </td>
            <td className="py-3 text-xs text-muted-foreground">
              {review.diff_stat && (
                <span>
                  {review.diff_stat.files_changed} 文件
                  <span className="text-green-500 ml-1">+{review.diff_stat.additions}</span>
                  <span className="text-red-500 ml-1">-{review.diff_stat.deletions}</span>
                </span>
              )}
            </td>
            <td className="py-3 text-xs text-muted-foreground">
              {new Date(review.created_at).toLocaleDateString('zh-CN')}
            </td>
            <td className="py-3"><ArrowRight className="w-4 h-4 text-muted-foreground" /></td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export function ReviewListPage() {
  const navigate = useNavigate();
  const { toast } = useToast();
  const [activeTab, setActiveTab] = useState<TabKey>('pending');

  // Per-tab state
  const [tabData, setTabData] = useState<Record<TabKey, { reviews: Review[]; total: number; page: number; loading: boolean }>>({
    pending: { reviews: [], total: 0, page: 1, loading: true },
    approved: { reviews: [], total: 0, page: 1, loading: true },
    rejected: { reviews: [], total: 0, page: 1, loading: true },
    all: { reviews: [], total: 0, page: 1, loading: true },
  });

  const fetchTab = useCallback((tab: TabKey, page: number) => {
    setTabData((prev) => ({ ...prev, [tab]: { ...prev[tab], loading: true } }));
    const config = TAB_CONFIG.find((t) => t.key === tab)!;
    const params: Record<string, unknown> = { page, page_size: 20 };
    if (config.humanStatus) {
      params.human_status = config.humanStatus;
    }
    reviewApi.list(params)
      .then((data) => {
        setTabData((prev) => ({
          ...prev,
          [tab]: { reviews: data.list, total: data.total, page, loading: false },
        }));
      })
      .catch((err) => {
        toast({ title: '获取审查列表失败', description: (err as Error).message, variant: 'destructive' });
        setTabData((prev) => ({ ...prev, [tab]: { ...prev[tab], loading: false } }));
      });
  }, [toast]);

  // Fetch data when tab changes or on initial load
  useEffect(() => {
    fetchTab(activeTab, tabData[activeTab].page);
  }, [activeTab]); // eslint-disable-line react-hooks/exhaustive-deps

  const handlePageChange = (tab: TabKey, newPage: number) => {
    setTabData((prev) => ({ ...prev, [tab]: { ...prev[tab], page: newPage } }));
    fetchTab(tab, newPage);
  };

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <ClipboardCheck className="w-6 h-6 text-primary" />
          审查列表
        </h1>
        <p className="text-muted-foreground mt-1">查看和管理代码审查记录</p>
      </div>

      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as TabKey)}>
        <TabsList>
          {TAB_CONFIG.map((tab) => (
            <TabsTrigger key={tab.key} value={tab.key}>
              {tab.label}
              {tabData[tab.key].total > 0 && !tabData[tab.key].loading && (
                <span className="ml-1.5 text-xs text-muted-foreground">
                  {tabData[tab.key].total}
                </span>
              )}
            </TabsTrigger>
          ))}
        </TabsList>

        {TAB_CONFIG.map((tab) => (
          <TabsContent key={tab.key} value={tab.key}>
            <Card>
              <CardContent className="pt-6">
                {tabData[tab.key].loading ? (
                  <Loading />
                ) : (
                  <ReviewTable reviews={tabData[tab.key].reviews} navigate={navigate} />
                )}
              </CardContent>
            </Card>

            {tabData[tab.key].total > 20 && (
              <div className="flex items-center justify-center gap-2 mt-4">
                <Button variant="outline" size="sm" disabled={tabData[tab.key].page === 1}
                  onClick={() => handlePageChange(tab.key, tabData[tab.key].page - 1)}>上一页</Button>
                <span className="text-sm text-muted-foreground">第 {tabData[tab.key].page} 页</span>
                <Button variant="outline" size="sm" disabled={tabData[tab.key].page * 20 >= tabData[tab.key].total}
                  onClick={() => handlePageChange(tab.key, tabData[tab.key].page + 1)}>下一页</Button>
              </div>
            )}
          </TabsContent>
        ))}
      </Tabs>
    </div>
  );
}

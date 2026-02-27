import { useState, useEffect, useCallback } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { requirementApi } from '@/api/requirement';
import { useToast } from '@/hooks/use-toast';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Loading } from '@/components/ui/spinner';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { FileText, ArrowRight, Search, AlertTriangle } from 'lucide-react';
import type { Requirement } from '@/types';

type TabKey = 'all' | 'created' | 'assigned';

const TAB_CONFIG: { key: TabKey; label: string }[] = [
  { key: 'all', label: '全部需求' },
  { key: 'created', label: '我创建的' },
  { key: 'assigned', label: '需要我做的' },
];

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

function RequirementTable({ requirements, navigate }: { requirements: Requirement[]; navigate: (path: string) => void }) {
  if (requirements.length === 0) {
    return <p className="text-sm text-muted-foreground text-center py-8">暂无需求</p>;
  }

  return (
    <table className="w-full">
      <thead>
        <tr className="border-b text-left text-sm text-muted-foreground">
          <th className="pb-2 font-medium">标题</th>
          <th className="pb-2 font-medium w-24">项目</th>
          <th className="pb-2 font-medium w-20">优先级</th>
          <th className="pb-2 font-medium w-24">状态</th>
          <th className="pb-2 font-medium w-28">截止时间</th>
          <th className="pb-2 font-medium w-16">创建者</th>
          <th className="pb-2 font-medium w-16">指派人</th>
          <th className="pb-2 font-medium w-10"></th>
        </tr>
      </thead>
      <tbody>
        {requirements.map((req) => (
          <tr key={req.id} className="border-b last:border-0 hover:bg-muted/50 cursor-pointer"
            onClick={() => navigate(`/requirements/${req.id}`)}>
            <td className="py-3 text-sm">
              <span className="inline-flex items-center gap-1.5">
                <span className="text-muted-foreground shrink-0">#{req.id}</span>
                {req.title}
                {req.doc_links && req.doc_links.length > 0 && (
                  <span title="已关联文档"><FileText className="w-3.5 h-3.5 text-primary/60 shrink-0" /></span>
                )}
              </span>
            </td>
            <td className="py-3 text-sm text-muted-foreground">{req.project?.name || '-'}</td>
            <td className="py-3"><Badge variant={priorityVariant(req.priority)} className="text-xs">{req.priority.toUpperCase()}</Badge></td>
            <td className="py-3"><Badge variant={statusVariant(req.status)} className="text-xs">{statusLabel[req.status] || req.status}</Badge></td>
            <td className="py-3 text-sm text-muted-foreground">
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
              ) : '-'}
            </td>
            <td className="py-3">
              {req.creator ? (
                req.creator.avatar ? (
                  <img src={req.creator.avatar} alt={req.creator.name} title={req.creator.name} className="w-7 h-7 rounded-full" />
                ) : (
                  <div title={req.creator.name} className="w-7 h-7 rounded-full bg-primary/20 flex items-center justify-center text-xs font-medium">
                    {req.creator.name[0]}
                  </div>
                )
              ) : <span className="text-sm text-muted-foreground">-</span>}
            </td>
            <td className="py-3">
              {req.assignee ? (
                req.assignee.avatar ? (
                  <img src={req.assignee.avatar} alt={req.assignee.name} title={req.assignee.name} className="w-7 h-7 rounded-full" />
                ) : (
                  <div title={req.assignee.name} className="w-7 h-7 rounded-full bg-primary/20 flex items-center justify-center text-xs font-medium">
                    {req.assignee.name[0]}
                  </div>
                )
              ) : <span className="text-sm text-muted-foreground">-</span>}
            </td>
            <td className="py-3"><ArrowRight className="w-4 h-4 text-muted-foreground" /></td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function TabContent({ scope, keyword }: { scope: string; keyword: string }) {
  const navigate = useNavigate();
  const { toast } = useToast();
  const [requirements, setRequirements] = useState<Requirement[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchRequirements = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, unknown> = { scope, page_size: 50 };
      if (keyword) params.keyword = keyword;
      const data = await requirementApi.list(params);
      setRequirements(data.list);
    } catch (err) {
      toast({ title: '获取需求失败', description: (err as Error).message, variant: 'destructive' });
    } finally {
      setLoading(false);
    }
  }, [scope, keyword]);

  useEffect(() => { fetchRequirements(); }, [fetchRequirements]);

  if (loading) return <Loading />;

  return <RequirementTable requirements={requirements} navigate={navigate} />;
}

export function RequirementListPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const activeTab = (searchParams.get('tab') as TabKey) || 'all';
  const [keyword, setKeyword] = useState('');

  const handleTabChange = (tab: string) => {
    setSearchParams({ tab });
  };

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <FileText className="w-6 h-6 text-primary" />
          需求列表
        </h1>
        <div className="relative w-64">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="搜索需求..."
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            className="pl-9"
          />
        </div>
      </div>

      <Card>
        <CardContent className="pt-6">
          <Tabs value={activeTab} onValueChange={handleTabChange}>
            <TabsList>
              {TAB_CONFIG.map((tab) => (
                <TabsTrigger key={tab.key} value={tab.key}>{tab.label}</TabsTrigger>
              ))}
            </TabsList>
            {TAB_CONFIG.map((tab) => (
              <TabsContent key={tab.key} value={tab.key}>
                <TabContent scope={tab.key} keyword={keyword} />
              </TabsContent>
            ))}
          </Tabs>
        </CardContent>
      </Card>
    </div>
  );
}

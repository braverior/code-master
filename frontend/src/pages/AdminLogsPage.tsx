import { useState, useEffect } from 'react';
import { adminApi } from '@/api/admin';
import { useToast } from '@/hooks/use-toast';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';

import { Badge } from '@/components/ui/badge';
import { Loading } from '@/components/ui/spinner';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import { FileText } from 'lucide-react';
import type { OperationLog } from '@/types';

const actionLabel: Record<string, string> = {
  create_project: '创建项目', update_project: '更新项目', archive_project: '归档项目',
  add_member: '添加成员', remove_member: '移除成员',
  create_repo: '关联仓库', delete_repo: '解除仓库',
  create_requirement: '创建需求', update_requirement: '更新需求', delete_requirement: '删除需求',
  generate_code: '代码生成', cancel_codegen: '取消生成',
  ai_review: 'AI 审查', human_review: '人工审查',
  review_approve: '审查通过', review_reject: '审查拒绝',
  create_mr: '创建 MR',
  update_role: '修改角色', update_status: '修改状态',
};

export function AdminLogsPage() {
  const { toast } = useToast();
  const [logs, setLogs] = useState<OperationLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [resourceType, setResourceType] = useState('all');

  useEffect(() => {
    setLoading(true);
    const params: Record<string, unknown> = { page, page_size: 20 };
    if (resourceType !== 'all') params.resource_type = resourceType;
    adminApi.getOperationLogs(params)
      .then((data) => {
        setLogs(data.list);
        setTotal(data.total);
      })
      .catch((err) => {
        toast({ title: '获取日志失败', description: (err as Error).message, variant: 'destructive' });
      })
      .finally(() => setLoading(false));
  }, [page, resourceType]);

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <FileText className="w-6 h-6 text-primary" />
          操作日志
        </h1>
        <p className="text-muted-foreground mt-1">系统操作审计日志</p>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3">
        <Select value={resourceType} onValueChange={(v) => { setResourceType(v); setPage(1); }}>
          <SelectTrigger className="w-36"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部类型</SelectItem>
            <SelectItem value="project">项目</SelectItem>
            <SelectItem value="requirement">需求</SelectItem>
            <SelectItem value="codegen">代码生成</SelectItem>
            <SelectItem value="review">审查</SelectItem>
          </SelectContent>
        </Select>
        <span className="text-sm text-muted-foreground ml-auto">共 {total} 条记录</span>
      </div>

      <Card>
        <CardContent className="pt-6">
          {loading ? <Loading /> : logs.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-8">暂无操作日志</p>
          ) : (
            <table className="w-full">
              <thead>
                <tr className="border-b text-left text-sm text-muted-foreground">
                  <th className="pb-3 font-medium w-28">操作人</th>
                  <th className="pb-3 font-medium">操作</th>
                  <th className="pb-3 font-medium w-24">资源类型</th>
                  <th className="pb-3 font-medium">详情</th>
                  <th className="pb-3 font-medium w-28">IP</th>
                  <th className="pb-3 font-medium w-36">时间</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((log) => (
                  <tr key={log.id} className="border-b last:border-0 hover:bg-muted/50">
                    <td className="py-3 text-sm">{log.user.name}</td>
                    <td className="py-3">
                      <Badge variant="outline" className="text-xs">
                        {actionLabel[log.action] || log.action}
                      </Badge>
                    </td>
                    <td className="py-3 text-sm text-muted-foreground">{log.resource_type}</td>
                    <td className="py-3 text-sm text-muted-foreground truncate max-w-[200px]">
                      {log.detail ? JSON.stringify(log.detail) : '-'}
                    </td>
                    <td className="py-3 text-xs text-muted-foreground font-mono">{log.ip}</td>
                    <td className="py-3 text-sm text-muted-foreground">
                      {new Date(log.created_at).toLocaleString('zh-CN')}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>

      {total > 20 && (
        <div className="flex items-center justify-center gap-2">
          <Button variant="outline" size="sm" disabled={page === 1} onClick={() => setPage(page - 1)}>上一页</Button>
          <span className="text-sm text-muted-foreground">第 {page} 页</span>
          <Button variant="outline" size="sm" disabled={page * 20 >= total} onClick={() => setPage(page + 1)}>下一页</Button>
        </div>
      )}
    </div>
  );
}

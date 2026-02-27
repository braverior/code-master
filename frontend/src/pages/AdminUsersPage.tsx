import { useState, useEffect } from 'react';
import { adminApi } from '@/api/admin';
import { useToast } from '@/hooks/use-toast';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Loading } from '@/components/ui/spinner';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Users, Search, MoreHorizontal, ShieldCheck, ShieldOff, UserCog, Ban, CheckCircle } from 'lucide-react';
import type { User } from '@/types';

const roleLabel: Record<string, string> = { pm: 'PM', rd: 'RD' };

export function AdminUsersPage() {
  const { toast } = useToast();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [keyword, setKeyword] = useState('');
  const [roleFilter, setRoleFilter] = useState('all');
  const [adminFilter, setAdminFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');

  const fetchUsers = async () => {
    setLoading(true);
    try {
      const params: Record<string, unknown> = { page, page_size: 20 };
      if (keyword) params.keyword = keyword;
      if (roleFilter !== 'all') params.role = roleFilter;
      if (adminFilter !== 'all') params.is_admin = adminFilter === 'true';
      if (statusFilter !== 'all') params.status = Number(statusFilter);
      const data = await adminApi.listUsers(params);
      setUsers(data.list);
      setTotal(data.total);
    } catch (err) {
      toast({ title: '获取用户列表失败', description: (err as Error).message, variant: 'destructive' });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchUsers(); }, [page, roleFilter, adminFilter, statusFilter]);

  const handleSearch = () => {
    setPage(1);
    fetchUsers();
  };

  const handleChangeRole = async (userId: number, role: string) => {
    try {
      await adminApi.updateUserRole(userId, role);
      toast({ title: '角色已更新', variant: 'success' });
      fetchUsers();
    } catch (err) {
      toast({ title: '更新失败', description: (err as Error).message, variant: 'destructive' });
    }
  };

  const handleToggleAdmin = async (userId: number, isAdmin: boolean) => {
    try {
      await adminApi.toggleUserAdmin(userId, isAdmin);
      toast({ title: isAdmin ? '已设为管理员' : '已取消管理员', variant: 'success' });
      fetchUsers();
    } catch (err) {
      toast({ title: '操作失败', description: (err as Error).message, variant: 'destructive' });
    }
  };

  const handleToggleStatus = async (userId: number, currentStatus: number) => {
    try {
      await adminApi.updateUserStatus(userId, currentStatus === 1 ? 0 : 1);
      toast({ title: currentStatus === 1 ? '用户已禁用' : '用户已启用', variant: 'success' });
      fetchUsers();
    } catch (err) {
      toast({ title: '操作失败', description: (err as Error).message, variant: 'destructive' });
    }
  };

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <Users className="w-6 h-6 text-primary" />
          用户管理
        </h1>
        <p className="text-muted-foreground mt-1">管理系统用户和角色</p>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input placeholder="搜索姓名或邮箱..." className="pl-9" value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()} />
        </div>
        <Select value={roleFilter} onValueChange={(v) => { setRoleFilter(v); setPage(1); }}>
          <SelectTrigger className="w-32"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部角色</SelectItem>
            <SelectItem value="pm">PM</SelectItem>
            <SelectItem value="rd">RD</SelectItem>
          </SelectContent>
        </Select>
        <Select value={adminFilter} onValueChange={(v) => { setAdminFilter(v); setPage(1); }}>
          <SelectTrigger className="w-32"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部权限</SelectItem>
            <SelectItem value="true">管理员</SelectItem>
            <SelectItem value="false">非管理员</SelectItem>
          </SelectContent>
        </Select>
        <Select value={statusFilter} onValueChange={(v) => { setStatusFilter(v); setPage(1); }}>
          <SelectTrigger className="w-28"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="1">正常</SelectItem>
            <SelectItem value="0">禁用</SelectItem>
          </SelectContent>
        </Select>
        <span className="text-sm text-muted-foreground ml-auto">共 {total} 个用户</span>
      </div>

      {/* Users Table */}
      <Card>
        <CardContent className="pt-6">
          {loading ? <Loading /> : (
            <table className="w-full">
              <thead>
                <tr className="border-b text-left text-sm text-muted-foreground">
                  <th className="pb-3 font-medium">用户</th>
                  <th className="pb-3 font-medium w-24">角色</th>
                  <th className="pb-3 font-medium w-20">状态</th>
                  <th className="pb-3 font-medium w-36">最后登录</th>
                  <th className="pb-3 font-medium w-36">注册时间</th>
                  <th className="pb-3 font-medium w-16">操作</th>
                </tr>
              </thead>
              <tbody>
                {users.map((u) => (
                  <tr key={u.id} className="border-b last:border-0 hover:bg-muted/50">
                    <td className="py-3">
                      <div className="flex items-center gap-3">
                        {u.avatar ? (
                          <img src={u.avatar} className="w-8 h-8 rounded-full" alt="" />
                        ) : (
                          <div className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center text-xs font-medium">
                            {u.name[0]}
                          </div>
                        )}
                        <div>
                          <p className="text-sm font-medium">{u.name}</p>
                          <p className="text-xs text-muted-foreground">{u.email}</p>
                        </div>
                      </div>
                    </td>
                    <td className="py-3">
                      <div className="flex items-center gap-1.5">
                        <Badge variant="secondary" className="text-xs">
                          {roleLabel[u.role] || u.role}
                        </Badge>
                        {u.is_admin && (
                          <Badge variant="default" className="text-xs px-1.5">
                            <ShieldCheck className="w-3 h-3" />
                          </Badge>
                        )}
                      </div>
                    </td>
                    <td className="py-3">
                      <Badge variant={u.status === 1 ? 'success' : 'destructive'} className="text-xs">
                        {u.status === 1 ? '正常' : '禁用'}
                      </Badge>
                    </td>
                    <td className="py-3 text-sm text-muted-foreground">
                      {u.last_login_at ? new Date(u.last_login_at).toLocaleString('zh-CN') : '-'}
                    </td>
                    <td className="py-3 text-sm text-muted-foreground">
                      {new Date(u.created_at).toLocaleDateString('zh-CN')}
                    </td>
                    <td className="py-3">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon" className="h-8 w-8">
                            <MoreHorizontal className="w-4 h-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => handleChangeRole(u.id, 'pm')}>
                            <UserCog className="w-4 h-4 mr-2" />设为 PM
                          </DropdownMenuItem>
                          <DropdownMenuItem onClick={() => handleChangeRole(u.id, 'rd')}>
                            <UserCog className="w-4 h-4 mr-2" />设为 RD
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          {u.is_admin ? (
                            <DropdownMenuItem onClick={() => handleToggleAdmin(u.id, false)}>
                              <ShieldOff className="w-4 h-4 mr-2" />取消管理员
                            </DropdownMenuItem>
                          ) : (
                            <DropdownMenuItem onClick={() => handleToggleAdmin(u.id, true)}>
                              <ShieldCheck className="w-4 h-4 mr-2" />设为管理员
                            </DropdownMenuItem>
                          )}
                          <DropdownMenuSeparator />
                          <DropdownMenuItem onClick={() => handleToggleStatus(u.id, u.status)}>
                            {u.status === 1 ? (
                              <><Ban className="w-4 h-4 mr-2" />禁用</>
                            ) : (
                              <><CheckCircle className="w-4 h-4 mr-2" />启用</>
                            )}
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
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

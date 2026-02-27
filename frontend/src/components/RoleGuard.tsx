import { Navigate } from 'react-router-dom';
import { useAuth } from '@/hooks/use-auth';

interface RoleGuardProps {
  children: React.ReactNode;
  roles: string[];
}

export function RoleGuard({ children, roles }: RoleGuardProps) {
  const { user } = useAuth();

  if (!user) {
    return <Navigate to="/" replace />;
  }

  // Check admin access
  if (roles.includes('admin') && user.is_admin) {
    return <>{children}</>;
  }

  // Check business role
  if (roles.includes(user.role)) {
    return <>{children}</>;
  }

  return <Navigate to="/" replace />;
}

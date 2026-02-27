// ============ Common ============

export interface PaginatedResponse<T> {
  list: T[];
  total: number;
  page: number;
  page_size: number;
}

export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

// ============ User / Auth ============

export interface User {
  id: number;
  name: string;
  avatar: string;
  email: string;
  role: 'pm' | 'rd';
  is_admin: boolean;
  status: number;
  is_new_user?: boolean;
  last_login_at: string;
  created_at: string;
}

export interface AuthTokenResponse {
  token: string;
  expire_at: string;
}

// ============ Project ============

export interface DocLink {
  title: string;
  url: string;
  type?: 'prd' | 'tech' | 'design' | 'other';
}

export interface ProjectMember {
  id: number;
  name: string;
  role: string;
  avatar: string;
  joined_at?: string;
}

export interface ProjectStats {
  total_requirements: number;
  draft: number;
  generating: number;
  generated: number;
  reviewing: number;
  approved: number;
  merged: number;
}

export interface Project {
  id: number;
  name: string;
  description: string;
  doc_links?: DocLink[];
  owner: { id: number; name: string; avatar: string };
  members?: ProjectMember[];
  repositories?: Repository[];
  stats?: ProjectStats;
  member_count?: number;
  repo_count?: number;
  requirement_count?: number;
  open_requirement_count?: number;
  status: 'active' | 'archived';
  created_at: string;
  updated_at: string;
}

// ============ Repository ============

export interface Repository {
  id: number;
  name: string;
  git_url: string;
  platform: 'gitlab' | 'github';
  platform_project_id?: string;
  default_branch: string;
  analysis_status: 'pending' | 'running' | 'completed' | 'failed';
  analysis_error?: string;
  analysis_result?: AnalysisResult;
  analyzed_at: string | null;
  project?: { id: number; name: string };
  created_at: string;
}

export interface AnalysisResult {
  modules: { path: string; description: string; files_count: number }[];
  tech_stack: string[];
  entry_points: string[];
  directory_structure: string;
  code_style: Record<string, string>;
}

export interface ConnectionTestResult {
  connected: boolean;
  branches?: string[];
  permissions?: {
    read: boolean;
    push: boolean;
  };
  error?: string;
}

// ============ Requirement ============

export type RequirementStatus = 'draft' | 'generating' | 'generated' | 'reviewing' | 'approved' | 'merged' | 'rejected';
export type Priority = 'p0' | 'p1' | 'p2' | 'p3';

export interface Requirement {
  id: number;
  project?: { id: number; name: string };
  title: string;
  description: string;
  doc_links?: DocLink[];
  doc_content_status?: string;
  priority: Priority;
  status: RequirementStatus;
  deadline?: string | null;
  creator: { id: number; name: string; avatar: string };
  assignee: { id: number; name: string; avatar: string } | null;
  repository: { id: number; name: string; platform?: string; git_url?: string } | null;
  codegen_tasks?: CodeGenTask[];
  latest_codegen?: { id: number; status: string; created_at: string } | null;
  latest_review?: { id: number; ai_score: number; human_status: string } | null;
  created_at: string;
  updated_at: string;
}

// ============ CodeGen ============

export type CodeGenStatus = 'pending' | 'cloning' | 'running' | 'completed' | 'failed' | 'cancelled';

export interface DiffStat {
  files_changed: number;
  additions: number;
  deletions: number;
  files?: DiffFile[];
}

export interface DiffFile {
  path: string;
  status: 'added' | 'modified' | 'deleted';
  language?: string;
  additions: number;
  deletions: number;
  diff?: string;
}

export interface CodeGenTask {
  id: number;
  requirement?: { id: number; title: string };
  repository?: { id: number; name: string; platform: string; git_url: string };
  source_branch: string;
  target_branch: string;
  status: CodeGenStatus;
  extra_context?: string;
  prompt?: string;
  diff_stat?: DiffStat;
  commit_sha?: string;
  error_message?: string;
  claude_cost_usd?: number;
  review?: {
    id: number;
    ai_score: number;
    ai_status: string;
    human_status: string;
  };
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
}

// ============ SSE Events ============

export interface SSEStatusEvent {
  status: CodeGenStatus;
  message?: string;
  pid?: number;
  files_changed?: number;
  additions?: number;
  deletions?: number;
}

export interface SSEOutputEvent {
  type: 'thinking' | 'text' | 'tool_use' | 'tool_result';
  content?: string;
  tool?: string;
  input?: Record<string, unknown>;
  id?: string;
  summary?: string;
  output?: string;
  exit_code?: number;
}

export interface SSELogEvent {
  level: 'info' | 'warn' | 'error';
  phase: 'clone' | 'claude' | 'push';
  message: string;
  detail?: Record<string, unknown>;
}

// Unified timeline entry for the output stream
export type StreamEntry =
  | { kind: 'output'; data: SSEOutputEvent }
  | { kind: 'log'; data: SSELogEvent };

export interface SSEProgressEvent {
  files_read: number;
  files_written: number;
  files_edited: number;
  turns_used: number;
  max_turns: number;
  current_action: string;
}

export interface SSEDoneEvent {
  task_id: number;
  status: string;
  review_id?: number;
}

// ============ Review ============

export interface ReviewIssue {
  severity: 'error' | 'warning' | 'info';
  file: string;
  line: number;
  code_snippet: string;
  message: string;
  suggestion: string;
}

export interface ReviewCategory {
  status: 'passed' | 'warning' | 'failed';
  details: string;
}

export interface AIReview {
  score: number;
  summary: string;
  issues: ReviewIssue[];
  categories: Record<string, ReviewCategory>;
}

export interface Review {
  id: number;
  codegen_task_id: number;
  requirement?: { id: number; title: string };
  project?: { id: number; name: string };
  repository?: { id: number; name: string };
  creator?: { id: number; name: string; avatar: string };
  ai_review: AIReview | null;
  ai_summary?: string;
  ai_score: number | null;
  ai_status: 'pending' | 'running' | 'passed' | 'failed';
  reviewers?: { id: number; name: string; avatar?: string }[];
  human_reviewer: { id: number; name: string; avatar?: string } | null;
  human_comment: string | null;
  human_status: 'pending' | 'approved' | 'rejected' | 'needs_revision';
  merge_request_url: string | null;
  merge_status: 'none' | 'created' | 'merged' | 'closed';
  source_branch?: string;
  target_branch?: string;
  git_url?: string;
  platform?: string;
  diff_stat?: DiffStat;
  created_at: string;
  updated_at: string;
}

// ============ Dashboard ============

export interface DashboardStats {
  my_projects: number;
  my_open_requirements: number;
  my_pending_reviews: number;
  codegen_running: number;
  recent_activity: DashboardActivity[];
}

export interface DashboardActivity {
  type: string;
  requirement?: { id: number; title: string };
  project?: { id: number; name: string };
  reviewer?: { id: number; name: string };
  time: string;
}

export interface MyTasks {
  pending_generate: {
    requirement_id: number;
    title: string;
    project: { id: number; name: string };
    priority: Priority;
    created_at: string;
  }[];
  running_tasks: {
    task_id: number;
    requirement: { id: number; title: string };
    status: string;
    started_at: string;
  }[];
  pending_reviews: {
    review_id: number;
    requirement: { id: number; title: string };
    ai_score: number;
    created_at: string;
  }[];
}

// ============ Admin ============

export interface OperationLog {
  id: number;
  user: { id: number; name: string };
  action: string;
  resource_type: string;
  resource_id: number;
  detail: Record<string, unknown>;
  ip: string;
  created_at: string;
}

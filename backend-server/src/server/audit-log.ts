import { getRepositories } from './repositories';
import type { AdminAuditEntry } from './repositories/types';

export async function writeAdminAudit(entry: AdminAuditEntry): Promise<void> {
  const repos = getRepositories();
  await repos.adminAudit.insert(entry);
}

// A copy of the backend's permission table, used ONLY to decide which buttons
// to show. The backend checks every request itself, so hiding a button is a
// convenience and never the protection.
//
// Keep this in step with backend/internal/permissions/permissions.go.

export type Role = 'viewer' | 'member' | 'admin' | 'owner'

export type Permission =
  | 'workspace:view'
  | 'workspace:update'
  | 'workspace:delete'
  | 'workspace:leave'
  | 'members:view'
  | 'members:invite'
  | 'members:manage'
  | 'project:view'
  | 'project:manage'
  | 'task:view'
  | 'task:write'
  | 'comment:write'
  | 'comment:moderate'

const rank: Record<Role, number> = { viewer: 1, member: 2, admin: 3, owner: 4 }

// For each permission, the lowest role that has it.
export const minimumRole: Record<Permission, Role> = {
  'workspace:view': 'viewer',
  'workspace:update': 'admin',
  'workspace:delete': 'owner',
  'workspace:leave': 'viewer',
  'members:view': 'viewer',
  'members:invite': 'admin',
  'members:manage': 'admin',
  'project:view': 'viewer',
  'project:manage': 'admin',
  'task:view': 'viewer',
  'task:write': 'member',
  'comment:write': 'member',
  'comment:moderate': 'admin',
}

/** Whether someone with this role may do this. Unknown or missing roles may do nothing. */
export function can(role: Role | undefined, permission: Permission): boolean {
  if (!role || !(role in rank)) return false
  return rank[role] >= rank[minimumRole[permission]]
}


export const allRoles: Role[] = ['viewer', 'member', 'admin', 'owner']

/** The roles that can be given in an invitation. Nobody is invited straight in as an owner. */
export type InviteRole = Exclude<Role, 'owner'>

/**
 * The roles `actor` may give to someone who currently has the role `target`.
 * An empty list means "you may not change this person". Mirrors the backend:
 * owners can do anything; admins only deal with people below them, and only
 * hand out roles below their own.
 */
export function rolesYouCanGrant(actor: Role, target: Role): Role[] {
  if (!can(actor, 'members:manage')) return []
  if (actor === 'owner') return allRoles
  return rank[target] < rank[actor] ? allRoles.filter((role) => rank[role] < rank[actor]) : []
}

/** Whether `actor` may remove someone whose role is `target`. */
export function canRemoveMember(actor: Role, target: Role): boolean {
  if (!can(actor, 'members:manage')) return false
  return actor === 'owner' || rank[target] < rank[actor]
}

/** The roles `actor` may invite people as. */
export function rolesYouCanInvite(actor: Role): InviteRole[] {
  if (!can(actor, 'members:invite')) return []
  const invitable: InviteRole[] = ['viewer', 'member', 'admin']
  return actor === 'owner' ? invitable : invitable.filter((role) => rank[role] < rank[actor])
}
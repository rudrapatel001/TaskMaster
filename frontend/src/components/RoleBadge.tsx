import type { Role } from '../lib/permissions'

const styles: Record<Role, string> = {
  owner: 'bg-amber-500/15 text-amber-300',
  admin: 'bg-indigo-500/15 text-indigo-300',
  member: 'bg-emerald-500/15 text-emerald-300',
  viewer: 'bg-slate-500/20 text-slate-300',
}

/** A small label showing a role, like "Owner". */
export function RoleBadge({ role }: { role: Role }) {
  return (
    <span className={`rounded-full px-2 py-0.5 text-xs font-medium capitalize ${styles[role]}`}>{role}</span>
  )
}
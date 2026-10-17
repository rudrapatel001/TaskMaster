import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './api'
import type { Role } from './permissions'
import { workspacesKey } from './workspaces'
import type { Workspace } from './workspaces'

export interface Member {
  user_id: string
  name: string
  email: string
  role: Role
  joined_at: string
}

export interface Invitation {
  id: string
  email: string
  role: Role
  invited_by: string
  created_at: string
  expires_at: string
}

/** A new invitation. The token (and so the link) exists only in this response. */
export interface CreatedInvitation extends Invitation {
  token: string
}

/** What someone holding an invitation link may see before logging in. */
export interface InvitationPreview {
  workspace_name: string
  email: string
  role: Role
  invited_by: string
  expires_at: string
}

const membersKey = (workspaceId: string) => ['workspaces', workspaceId, 'members'] as const
const invitationsKey = (workspaceId: string) => ['workspaces', workspaceId, 'invitations'] as const

export function useMembers(workspaceId: string) {
  return useQuery({
    queryKey: membersKey(workspaceId),
    queryFn: () => api<Member[]>(`/workspaces/${workspaceId}/members`),
  })
}

export function useChangeRole(workspaceId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: Role }) =>
      api<Member>(`/workspaces/${workspaceId}/members/${userId}`, { method: 'PATCH', body: { role } }),
    onSuccess: (updated) => {
      queryClient.setQueryData<Member[]>(membersKey(workspaceId), (old) =>
        (old ?? []).map((m) => (m.user_id === updated.user_id ? updated : m)),
      )
    },
  })
}

export function useRemoveMember(workspaceId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) =>
      api<void>(`/workspaces/${workspaceId}/members/${userId}`, { method: 'DELETE' }),
    onSuccess: (_result, userId) => {
      queryClient.setQueryData<Member[]>(membersKey(workspaceId), (old) =>
        (old ?? []).filter((m) => m.user_id !== userId),
      )
    },
  })
}

export function useInvitations(workspaceId: string) {
  return useQuery({
    queryKey: invitationsKey(workspaceId),
    queryFn: () => api<Invitation[]>(`/workspaces/${workspaceId}/invitations`),
  })
}

export function useCreateInvitation(workspaceId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { email: string; role: Role }) =>
      api<CreatedInvitation>(`/workspaces/${workspaceId}/invitations`, { method: 'POST', body: input }),
    // Inviting the same email again replaces the old invitation, so just reload the list.
    onSuccess: () => queryClient.invalidateQueries({ queryKey: invitationsKey(workspaceId) }),
  })
}

export function useRevokeInvitation(workspaceId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (invitationId: string) =>
      api<void>(`/workspaces/${workspaceId}/invitations/${invitationId}`, { method: 'DELETE' }),
    onSuccess: (_result, invitationId) => {
      queryClient.setQueryData<Invitation[]>(invitationsKey(workspaceId), (old) =>
        (old ?? []).filter((i) => i.id !== invitationId),
      )
    },
  })
}

/** The public preview of an invitation link. It needs no login. */
export function useInvitationPreview(token: string) {
  return useQuery({
    queryKey: ['invitation', token],
    queryFn: () => api<InvitationPreview>(`/invitations/${token}`, { auth: false }),
  })
}

export function useAcceptInvitation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (token: string) => api<Workspace>(`/invitations/${token}/accept`, { method: 'POST' }),
    // Wait for the workspace list to reload, so the new workspace is there when we open it.
    onSuccess: () => queryClient.invalidateQueries({ queryKey: workspacesKey, exact: true }),
  })
}
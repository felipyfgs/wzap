export interface GroupParticipant {
  jid: string
  phoneNumber?: string
  lid?: string
  isAdmin: boolean
  isSuperAdmin: boolean
  displayName?: string
}

export interface JoinRequest {
  jid: string
  requestedAt?: string
}

export interface Subgroup {
  jid: string
  name?: string
}

export interface GroupDetail {
  jid: string
  name: string
  topic?: string
  isAdmin: boolean
  isParent: boolean
  isLocked: boolean
  isAnnounce: boolean
  joinApproval: boolean
  isEphemeral: boolean
  ephemeralTimer: number
  participants: GroupParticipant[]
  subgroups?: Subgroup[]
  createdAt?: string
}

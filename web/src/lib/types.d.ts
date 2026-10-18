export type RoomRole = 'host' | 'student';
export type FilterTab = 'popular' | 'latest' | 'pinned' | 'answered';
export type View = 'landing' | 'room';

export interface Question {
  id: string;
  roomId: string;
  authorAlias: string;
  authorId: string;
  authorRole: string;
  text: string;
  upvotes: number;
  status: 'unanswered' | 'pinned' | 'answered';
  createdAt: string;
  hasVoted: boolean;
}

export interface TypingUser { id: string; alias: string }
export interface TypingPayload { id: string; alias?: string; isTyping: boolean }
export interface PulseStats {
  paham: number;
  bingung: number;
  kecepatan: number;
  total: number;
  pahamPct: number;
  bingungPct: number;
  kecepatanPct: number;
}

export interface RoomState {
  roomId: string;
  roomName: string;
  isHost: boolean;
  hostKey: string;
  userAlias: string;
  isFrozen: boolean;
  questions: Question[];
  votedQuestionIds: string[];
  typingUsers: TypingUser[];
  pulseStats: PulseStats;
  activeUsers: number;
  currentView: View;
  filterTab: FilterTab;
  isProjectorOpen: boolean;
  sessionId: string;
}

export interface RoomSnapshot {
  participantId?: string;
  participantToken?: string;
  id: string;
  name?: string;
  isHost?: boolean;
  userAlias?: string;
  isFrozen?: boolean;
  questions?: Question[];
  votedQuestionIds?: string[];
  typingUsers?: TypingPayload[];
  pulseStats?: PulseStats;
  activeUsers?: number;
}

export interface PublicRoom {
  id: string;
  name: string;
  questionCount: number;
  activeUsers: number;
  createdAt: string;
}

export type ServerEvent =
  | { type: 'ROOM_STATE'; payload: RoomSnapshot }
  | { type: 'TYPING_UPDATED'; payload: TypingPayload }
  | { type: 'QUESTION_ADDED' | 'QUESTION_UPDATED'; payload: Question }
  | { type: 'QUESTION_ACCEPTED'; requestId: string }
  | { type: 'VOTE_UPDATED'; payload: { questionId: string; hasVoted: boolean; upvotes: number } }
  | { type: 'QUESTION_DELETED'; payload: { questionId: string } }
  | { type: 'PULSE_UPDATED'; payload: PulseStats }
  | { type: 'ROOM_FROZEN'; payload: { isFrozen: boolean } }
  | { type: 'USER_COUNT'; payload: { count: number } }
  | { type: 'ERROR'; payload: string; requestId?: string }
  | { type: 'TOAST' | 'ROOM_CLOSED' | 'IDENTITY_EXPIRED'; payload: string };

export type ClientAction = 'TYPING' | 'SUBMIT_QUESTION' | 'UPVOTE_QUESTION' | 'UPDATE_PULSE'
  | 'HOST_CLOSE' | 'HOST_PIN' | 'HOST_RESOLVE' | 'HOST_DELETE' | 'HOST_FREEZE' | 'HOST_CLEAR_PULSE';
export interface ActionPayload {
  text?: string;
  questionId?: string;
  requestId?: string;
  hostKey?: string;
  vote?: 'paham' | 'bingung' | 'kecepatan';
}

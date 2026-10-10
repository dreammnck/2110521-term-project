export type Role = "staff" | "resident";

export interface User {
  id: string;
  email: string;
  name: string;
  role: Role;
  roomId?: string;
  roomNumber?: string;
  buildingId?: string;
  building?: string;
  floor?: number;
}

export interface Room {
  id: string;
  number: string;
  floor: number;
  buildingId: string;
  building: string;
}

export type ParcelStatus = "WAITING" | "PICKED_UP";

export interface Parcel {
  id: string;
  roomId: string;
  carrier: string;
  trackingNo: string;
  note: string;
  hasPhoto: boolean;
  status: ParcelStatus;
  receivedAt: string;
  pickedUpAt?: string;
  pickedUpBy?: string;
  registeredBy: string;
}

export type BillStatus = "UNPAID" | "PENDING" | "PAID" | "OVERDUE";

export interface Bill {
  id: string;
  roomId: string;
  cycle: string;
  waterUnits: number;
  waterAmount: number;
  electricUnits: number;
  electricAmount: number;
  total: number;
  dueDate: string;
  status: BillStatus;
  issuedAt: string;
}

export interface BillInput {
  roomId: string;
  cycle: string;
  waterUnits: number;
  electricUnits: number;
  dueDate: string;
}

export interface Receipt {
  receiptNo: string;
  paymentReference: string;
  paidAt: string;
  amount: number;
  bill: Bill;
}

export type TargetType = "ALL" | "BUILDING" | "FLOOR";
export type AnnouncementStatus = "DRAFT" | "PUBLISHED" | "ARCHIVED";

export interface Announcement {
  id: string;
  title: string;
  body: string;
  category: string;
  status: AnnouncementStatus;
  targetType: TargetType;
  targetBuildingId?: string;
  targetFloor?: number;
  publishAt: string;
  createdBy: string;
  updatedAt: string;
  read: boolean;
}

export interface AnnouncementInput {
  title: string;
  body: string;
  category: string;
  targetType: TargetType;
  targetBuildingId?: string;
  targetFloor?: number;
}

export interface Notification {
  id: string;
  type: string;
  title: string;
  body: string;
  refType: string;
  refId: string;
  readAt?: string;
  createdAt: string;
}

export const ANNOUNCEMENT_CATEGORIES = ["GENERAL", "MAINTENANCE", "SAFETY", "EVENT", "BILLING"] as const;

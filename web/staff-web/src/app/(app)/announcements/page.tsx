"use client";

import { useState } from "react";
import { mutate } from "swr";
import { patch, post, useApi, useRooms } from "@condo/shared/hooks";
import { Async, Button, Card, ErrorNote, Field, Input, Notice, PageHeader, Select, StatusBadge, TextArea } from "@condo/shared/ui";
import { ANNOUNCEMENT_CATEGORIES, dateTime, describeTarget, type Announcement, type AnnouncementInput, type TargetType } from "@condo/shared";

const empty: AnnouncementInput = { title: "", body: "", category: "GENERAL", targetType: "ALL" };

function Editor({ editing, onClose, onSaved }: { editing: Announcement | null; onClose: () => void; onSaved: (msg: string) => void }) {
  const { rooms, buildings } = useRooms();
  const [f, setF] = useState<AnnouncementInput>(
    editing
      ? { title: editing.title, body: editing.body, category: editing.category, targetType: editing.targetType, targetBuildingId: editing.targetBuildingId, targetFloor: editing.targetFloor }
      : empty,
  );
  const [error, setError] = useState<unknown>();
  const [busy, setBusy] = useState(false);
  const floors = [...new Set((rooms ?? []).filter((r) => r.buildingId === f.targetBuildingId).map((r) => r.floor))].sort((a, b) => a - b);

  return (
    <Card className="mb-6">
      <h2 className="mb-3 font-semibold">{editing ? "Edit announcement" : "New announcement"}</h2>
      <form
        className="grid gap-3 sm:grid-cols-2"
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          setError(undefined);
          const body: AnnouncementInput = {
            ...f,
            targetBuildingId: f.targetType === "ALL" ? undefined : f.targetBuildingId,
            targetFloor: f.targetType === "FLOOR" ? f.targetFloor : undefined,
          };
          try {
            if (editing) await patch(`announcements/${editing.id}`, body);
            else await post("announcements", body);
            onSaved(editing ? "Announcement updated." : "Announcement published and residents notified.");
          } catch (err) {
            setError(err);
            setBusy(false);
          }
        }}
      >
        <div className="sm:col-span-2">
          <Field label="Title">
            <Input required maxLength={150} value={f.title} onChange={(e) => setF({ ...f, title: e.target.value })} />
          </Field>
        </div>
        <div className="sm:col-span-2">
          <Field label="Details">
            <TextArea required rows={5} value={f.body} onChange={(e) => setF({ ...f, body: e.target.value })} />
          </Field>
        </div>
        <Field label="Category">
          <Select value={f.category} onChange={(e) => setF({ ...f, category: e.target.value })}>
            {ANNOUNCEMENT_CATEGORIES.map((c) => (
              <option key={c}>{c}</option>
            ))}
          </Select>
        </Field>
        <Field label="Audience">
          <Select
            value={f.targetType}
            onChange={(e) => setF({ ...f, targetType: e.target.value as TargetType, targetBuildingId: undefined, targetFloor: undefined })}
          >
            <option value="ALL">All residents</option>
            <option value="BUILDING">A specific building</option>
            <option value="FLOOR">A specific floor</option>
          </Select>
        </Field>
        {f.targetType !== "ALL" && (
          <Field label="Building">
            <Select required value={f.targetBuildingId ?? ""} onChange={(e) => setF({ ...f, targetBuildingId: e.target.value, targetFloor: undefined })}>
              <option value="" disabled>
                Select a building…
              </option>
              {buildings.map((b) => (
                <option key={b.id} value={b.id}>
                  {b.name}
                </option>
              ))}
            </Select>
          </Field>
        )}
        {f.targetType === "FLOOR" && (
          <Field label="Floor">
            <Select required value={f.targetFloor ?? ""} onChange={(e) => setF({ ...f, targetFloor: Number(e.target.value) })}>
              <option value="" disabled>
                Select a floor…
              </option>
              {floors.map((n) => (
                <option key={n} value={n}>
                  Floor {n}
                </option>
              ))}
            </Select>
          </Field>
        )}
        <div className="space-y-3 sm:col-span-2">
          <ErrorNote error={error} />
          <div className="flex gap-2">
            <Button type="submit" disabled={busy}>
              {busy ? "Saving…" : editing ? "Save changes" : "Publish"}
            </Button>
            <Button type="button" variant="secondary" onClick={onClose}>
              Cancel
            </Button>
          </div>
        </div>
      </form>
    </Card>
  );
}

export default function AnnouncementsPage() {
  const { data, error } = useApi<Announcement[]>("announcements");
  const { buildings } = useRooms();
  const [editing, setEditing] = useState<Announcement | null | "new">(null);
  const [notice, setNotice] = useState("");
  const [actionError, setActionError] = useState<unknown>();
  const buildingName = (id?: string) => buildings.find((b) => b.id === id)?.name;
  const refresh = () => mutate("announcements");

  return (
    <>
      <PageHeader
        title="Announcements"
        subtitle="Publish notices to all residents or a specific building or floor"
        actions={!editing && <Button onClick={() => { setNotice(""); setEditing("new"); }}>New announcement</Button>}
      />
      {editing && (
        <Editor
          key={editing === "new" ? "new" : editing.id}
          editing={editing === "new" ? null : editing}
          onClose={() => setEditing(null)}
          onSaved={(m) => {
            setEditing(null);
            setNotice(m);
            refresh();
          }}
        />
      )}
      {notice && (
        <div className="mb-4">
          <Notice>{notice}</Notice>
        </div>
      )}
      <ErrorNote error={actionError} />
      <Async data={data} error={error} empty="No announcements yet.">
        {(items) => (
          <ul className="space-y-3">
            {items.map((a) => (
              <li key={a.id}>
                <Card>
                  <div className="flex flex-wrap items-start justify-between gap-2">
                    <div className="min-w-0">
                      <h3 className="font-semibold">{a.title}</h3>
                      <p className="text-xs text-slate-500">
                        {a.category} · {describeTarget(a, buildingName(a.targetBuildingId))} · {dateTime(a.publishAt)}
                      </p>
                    </div>
                    <StatusBadge status={a.status} />
                  </div>
                  <p className="mt-2 whitespace-pre-line text-sm text-slate-700">{a.body}</p>
                  {a.status !== "ARCHIVED" && (
                    <div className="mt-3 flex gap-2">
                      <Button variant="secondary" onClick={() => { setNotice(""); setEditing(a); }}>
                        Edit
                      </Button>
                      <Button
                        variant="ghost"
                        onClick={async () => {
                          setActionError(undefined);
                          try {
                            await post(`announcements/${a.id}/archive`);
                            refresh();
                          } catch (err) {
                            setActionError(err);
                          }
                        }}
                      >
                        Archive
                      </Button>
                    </div>
                  )}
                </Card>
              </li>
            ))}
          </ul>
        )}
      </Async>
    </>
  );
}

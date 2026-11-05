import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  api,
  type Character,
  type CharacterInput,
  type Relationship,
} from "./api";

const textFields = [
  "role",
  "pronouns",
  "description",
  "backstory",
  "motivation",
  "voice",
] as const;
const listFields = [
  "traits",
  "goals",
  "fears",
  "flaws",
  "values",
  "constraints",
  "secrets",
  "initialKnowledge",
] as const;
const labels: Record<string, string> = {
  initialKnowledge: "Initial knowledge (known to this character)",
  secrets: "Author secrets (not automatically known)",
  constraints: "Behavior constraints",
  voice: "Voice and speech patterns",
};
const empty = (): CharacterInput => ({
  name: "",
  traits: [],
  role: "",
  pronouns: "",
  description: "",
  backstory: "",
  motivation: "",
  voice: "",
  goals: [],
  fears: [],
  flaws: [],
  values: [],
  constraints: [],
  secrets: [],
  initialKnowledge: [],
  relationships: [],
});

// Send only authoring fields; API response metadata is not part of PUT input.
function toInput(value: Character): CharacterInput {
  const base = empty();
  return {
    name: value.name,
    traits: value.traits ?? [],
    ...Object.fromEntries(
      [...textFields, ...listFields, "relationships" as const].map((key) => [
        key,
        value[key] ?? base[key],
      ]),
    ),
  } as CharacterInput;
}

export function CharacterPanel({ storyID }: { storyID: string }) {
  const cache = useQueryClient();
  const [editing, setEditing] = useState<Character | null>(null);
  const [input, setInput] = useState<CharacterInput>(empty);
  const [notice, setNotice] = useState("");
  const characters = useQuery({
    queryKey: ["characters", storyID],
    queryFn: () => api.characters(storyID),
  });
  const save = useMutation({
    mutationFn: () =>
      api.saveCharacter(
        storyID,
        { ...input, expectedVersion: editing?.version },
        editing?.characterId,
      ),
    onSuccess: (value) => {
      setEditing(value);
      setInput(toInput(value));
      setNotice(
        `Saved version ${value.version}. Existing drafts must be regenerated.`,
      );
      cache.invalidateQueries({ queryKey: ["characters", storyID] });
    },
    onError: (error) => setNotice(error.message),
  });
  const select = (value: Character | null) => {
    setEditing(value);
    setInput(value ? toInput(value) : empty());
    setNotice("");
  };
  const relation = (index: number, patch: Partial<Relationship>) =>
    setInput((old) => ({
      ...old,
      relationships: old.relationships.map((value, i) =>
        i === index ? { ...value, ...patch } : value,
      ),
    }));
  return (
    <section className="character-panel">
      <span className="eyebrow">Character bible</span>
      <h2>People drive the story</h2>
      <p>
        Definitions are versioned. Initial knowledge is a baseline; accepted
        scenes add learned facts and emotions.
      </p>
      <select
        aria-label="Choose character"
        value={editing?.characterId ?? ""}
        onChange={(event) =>
          select(
            characters.data?.find(
              (value) => value.characterId === event.target.value,
            ) ?? null,
          )
        }
      >
        <option value="">New character</option>
        {characters.data?.map((value) => (
          <option key={value.characterId} value={value.characterId}>
            {value.name} · v{value.version}
          </option>
        ))}
      </select>
      {characters.error && <p role="alert">{characters.error.message}</p>}
      {notice && <p role="status">{notice}</p>}
      <form
        onSubmit={(event) => {
          event.preventDefault();
          save.mutate();
        }}
      >
        <label>
          Name
          <input
            required
            maxLength={200}
            value={input.name}
            onChange={(event) =>
              setInput({ ...input, name: event.target.value })
            }
          />
        </label>
        {textFields.map((field) => (
          <label key={field}>
            {labels[field] ?? field}
            <textarea
              aria-label={labels[field] ?? field}
              maxLength={8000}
              rows={field === "backstory" ? 4 : 2}
              value={input[field]}
              onChange={(event) =>
                setInput({ ...input, [field]: event.target.value })
              }
            />
          </label>
        ))}
        {listFields.map((field) => (
          <ListField
            key={`${editing?.characterId}-${editing?.version}-${field}`}
            label={labels[field] ?? field}
            value={input[field] ?? []}
            onChange={(value) =>
              setInput((old) => ({ ...old, [field]: value }))
            }
          />
        ))}
        <h3>Relationships</h3>
        {input.relationships?.map((value, index) => (
          <fieldset key={index}>
            <label>
              Other character
              <select
                required
                value={value.characterId}
                onChange={(event) =>
                  relation(index, { characterId: event.target.value })
                }
              >
                <option value="">Choose character</option>
                {characters.data
                  ?.filter((item) => item.characterId !== editing?.characterId)
                  .map((item) => (
                    <option key={item.characterId} value={item.characterId}>
                      {item.name}
                    </option>
                  ))}
              </select>
            </label>
            <label>
              Relationship kind
              <input
                required
                maxLength={100}
                placeholder="Ally, rival, sibling…"
                value={value.kind}
                onChange={(event) =>
                  relation(index, { kind: event.target.value })
                }
              />
            </label>
            <label>
              Dynamic
              <textarea
                maxLength={2000}
                value={value.description}
                onChange={(event) =>
                  relation(index, { description: event.target.value })
                }
              />
            </label>
            <button
              type="button"
              onClick={() =>
                setInput({
                  ...input,
                  relationships: input.relationships.filter(
                    (_, i) => i !== index,
                  ),
                })
              }
            >
              Remove relationship
            </button>
          </fieldset>
        ))}
        <button
          type="button"
          onClick={() =>
            setInput({
              ...input,
              relationships: [
                ...(input.relationships ?? []),
                { characterId: "", kind: "", description: "" },
              ],
            })
          }
        >
          Add relationship
        </button>
        <button className="save-action" disabled={save.isPending} type="submit">
          {save.isPending
            ? "Saving…"
            : editing
              ? "Save new version"
              : "Create character"}
        </button>
      </form>
    </section>
  );
}
function ListField({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string[];
  onChange: (value: string[]) => void;
}) {
  const [text, setText] = useState(value.join("\n"));
  return (
    <label>
      {label}
      <small>One per line · up to 32</small>
      <textarea
        aria-label={label}
        rows={3}
        value={text}
        onChange={(event) => {
          setText(event.target.value);
          onChange(
            event.target.value
              .split("\n")
              .map((item) => item.trim())
              .filter(Boolean),
          );
        }}
      />
    </label>
  );
}

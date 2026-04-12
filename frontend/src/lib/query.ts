export type ParsedQuery = {
  keywords: string[];
  tags: string[];
  contexts: string[];
  projects: string[];
  negativeKeywords: string[];
  negativeTags: string[];
  negativeContexts: string[];
  negativeProjects: string[];
  flags: { completed?: boolean; archived?: boolean; status?: string[] };
  priority?: string[];
  due?: { op: ':' | '<=' | '>='; date: string }[];
  t?:   { op: ':' | '<=' | '>='; date: string }[];
};

export function parseQuery(input: string): ParsedQuery {
  const out: ParsedQuery = { 
    keywords: [], 
    tags: [], 
    contexts: [], 
    projects: [], 
    negativeKeywords: [],
    negativeTags: [],
    negativeContexts: [],
    negativeProjects: [],
    flags: {} 
  };
  const tokens = input.match(/"[^"]+"|\S+/g) ?? [];
  for (const tok of tokens) {
    const t = tok.replace(/^"|"$/g, "");
    
    // Handle negative filters
    if (t.startsWith("-#")) out.negativeTags.push(t.slice(2));
    else if (t.startsWith("-@")) out.negativeContexts.push(t.slice(2));
    else if (t.startsWith("-+")) out.negativeProjects.push(t.slice(2));
    else if (t.startsWith("-") && t !== "-completed") out.negativeKeywords.push(t.slice(1));
    // Handle positive filters
    else if (t.startsWith("#")) out.tags.push(t.slice(1));
    else if (t.startsWith("@")) out.contexts.push(t.slice(1));
    else if (t.startsWith("+")) out.projects.push(t.slice(1));
    else if (/^pri(or(ity)?)?:[A-Z]$/i.test(t)) (out.priority ||= []).push(t.split(":")[1]!.toUpperCase());
    else if (/^(due|t)(:|<=|>=)\d{4}-\d{2}-\d{2}$/.test(t)) {
      const m = t.match(/^(due|t)(:|<=|>=)/)!;
      const key = m[1]!;
      const op = m[2]!;
      const date = t.slice(key.length + op.length);
      (out as any)[key] ??= [];
      (out as any)[key].push({ op: op as any, date });
    }
    else if (t === "-completed") out.flags.completed = false;
    else if (t === "completed") out.flags.completed = true;
    else if (t.startsWith("status:")) (out.flags.status ??= []).push(t.split(":")[1]!);
    else if (t === "is:archived") out.flags.archived = true;
    else out.keywords.push(t);
  }
  return out;
}

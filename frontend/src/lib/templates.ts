export type TaskTemplate = {
  id: string;
  name: string;
  contexts: string[];
  projects: string[];
  tags: string[];
  priority?: 'A' | 'B' | 'C' | undefined;
  createdAt: string;
};

const STORAGE_KEY = 'nil.taskTemplates';

export function getTemplates(): TaskTemplate[] {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored) return [];
    const parsed = JSON.parse(stored);
    return Array.isArray(parsed) ? parsed : [];
  } catch (err) {
    console.error('Failed to load templates:', err);
    return [];
  }
}

export function saveTemplate(template: Omit<TaskTemplate, 'id' | 'createdAt'>): TaskTemplate {
  const templates = getTemplates();
  const newTemplate: TaskTemplate = {
    ...template,
    id: Date.now().toString(),
    createdAt: new Date().toISOString(),
  };
  templates.push(newTemplate);
  localStorage.setItem(STORAGE_KEY, JSON.stringify(templates));
  return newTemplate;
}

export function updateTemplate(id: string, updates: Partial<TaskTemplate>): void {
  const templates = getTemplates();
  const index = templates.findIndex(t => t.id === id);
  const existing = templates[index];
  if (index >= 0 && existing) {
    templates[index] = { ...existing, ...updates };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(templates));
  }
}

export function deleteTemplate(id: string): void {
  const templates = getTemplates();
  const filtered = templates.filter(t => t.id !== id);
  localStorage.setItem(STORAGE_KEY, JSON.stringify(filtered));
}

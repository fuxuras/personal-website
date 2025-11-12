
import { defineCollection, z } from 'astro:content';

const blogCollection = defineCollection({
  type: 'content',
  schema: z.object({
    title: z.string(),
    description: z.string(),
    date: z.coerce.date(),
    draft: z.boolean().optional(),
    tags: z.array(z.string()),
    lang: z.string(),
    translation: z.string().optional(),
  }),
});

const projectCollection = defineCollection({
  type: 'content',
  schema: z.object({
    title: z.string(),
    description: z.string(),
    tags: z.array(z.string()),
    repoUrl: z.string().optional(),
    liveUrl: z.string().optional()
  }),
});

const commonplaceCollection = defineCollection({
  type: 'content',
  schema: z.object({
    text: z.string(),
    author: z.string().optional(),
    subtext: z.string().optional(),
    pubDate: z.coerce.date().optional(), // For sorting
  }),
});

export const collections = {
  'blog': blogCollection,
  'project': projectCollection,
  'commonplace': commonplaceCollection
};
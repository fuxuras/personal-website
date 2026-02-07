import React, { useEffect, useMemo, useState } from 'react';
import { getApiBase } from '../lib/api';

type FeedPost = {
  id: number;
  content: string;
  createdAt: string;
};

type FeedResponse = {
  posts: FeedPost[];
};

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat('en-US', {
    dateStyle: 'medium',
    timeStyle: 'short',
    hour12: false,
  }).format(date);
}

export default function FeedPreview() {
  const apiBase = useMemo(() => getApiBase(), []);
  const [posts, setPosts] = useState<FeedPost[]>([]);

  useEffect(() => {
    const load = async () => {
      try {
        const res = await fetch(`${apiBase}/api/feed?offset=0&limit=2`);
        if (!res.ok) return;
        const data = (await res.json()) as FeedResponse;
        setPosts(data.posts ?? []);
      } catch (error) {
        console.error('Failed to load feed preview', error);
      }
    };

    load();
  }, [apiBase]);

  return (
    <div className="w-[80%] mx-auto border border-border divide-y divide-border">
      {posts.map((post) => (
        <article key={post.id} className="p-5 space-y-3">
          <div className="caption">{formatDate(post.createdAt)}</div>
          <div
            className="feed-content text-sm leading-relaxed text-muted-foreground"
            dangerouslySetInnerHTML={{ __html: post.content }}
          />
        </article>
      ))}
      {posts.length === 0 && (
        <div className="p-5 text-sm text-muted-foreground">
          Duvar yüklenemedi.
        </div>
      )}
    </div>
  );
}

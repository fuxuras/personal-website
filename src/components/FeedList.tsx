import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { getApiBase } from '../lib/api';

type FeedPost = {
  id: number;
  title: string;
  content: string;
  createdAt: string;
};

type FeedResponse = {
  posts: FeedPost[];
  offset: number;
  limit: number;
  nextOffset: number;
  hasMore: boolean;
};

const LIMIT = 20;

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat('en-US', {
    dateStyle: 'medium',
    timeStyle: 'short',
    hour12: false,
  }).format(date);
}

export default function FeedList() {
  const apiBase = useMemo(() => getApiBase(), []);
  const [posts, setPosts] = useState<FeedPost[]>([]);
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const sentinelRef = useRef<HTMLDivElement | null>(null);
  const loadingRef = useRef(false);

  const loadMore = useCallback(async () => {
    if (loadingRef.current || !hasMore) return;
    loadingRef.current = true;
    setLoading(true);
    setError(null);

    try {
      const res = await fetch(
        `${apiBase}/api/feed?offset=${offset}&limit=${LIMIT}`
      );
      if (!res.ok) {
        throw new Error(`Failed to load feed (${res.status})`);
      }
      const data = (await res.json()) as FeedResponse;
      setPosts((prev) => [...prev, ...data.posts]);
      setOffset(data.nextOffset ?? offset + data.posts.length);
      setHasMore(data.hasMore ?? data.posts.length === LIMIT);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load feed');
    } finally {
      setLoading(false);
      loadingRef.current = false;
    }
  }, [apiBase, hasMore, offset]);

  useEffect(() => {
    loadMore();
  }, []);

  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          loadMore();
        }
      },
      { rootMargin: '200px' }
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [loadMore]);

  return (
    <div className="frame border-x border-border">
      <div>
        {posts.map((post, index) => (
          <article
            key={post.id}
            className="p-5 space-y-3 border-b border-border last:border-b-0"
          >
            <div className="flex flex-wrap items-start justify-between gap-3">
              <h2 className="text-base font-semibold tracking-tight">
                {post.title}
              </h2>
              <span className="caption">{formatDate(post.createdAt)}</span>
            </div>
            <p className="text-sm leading-relaxed text-muted-foreground whitespace-pre-line">
              {post.content}
            </p>
          </article>
        ))}

        {error && (
          <div className="p-4 text-sm text-destructive">
            {error}
          </div>
        )}
      </div>

      <div ref={sentinelRef} className="h-8"></div>

      <div className="border-t border-border p-4 text-xs font-mono uppercase tracking-[0.25em] text-muted-foreground">
        {loading ? 'Loading' : hasMore ? 'Scroll to load more' : 'End of feed'}
      </div>
    </div>
  );
}

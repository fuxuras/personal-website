import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { getApiBase } from '../lib/api';

type FeedPost = {
  id: number;
  content: string;
  createdAt: string;
  views: number;
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

function getViewerToken() {
  const tokenKey = 'viewer_token';
  let token = localStorage.getItem(tokenKey);
  if (!token) {
    token = crypto.randomUUID();
    localStorage.setItem(tokenKey, token);
  }
  return token;
}

function getViewedFeedSet() {
  const viewedKey = 'viewer_views_feed';
  const raw = localStorage.getItem(viewedKey);
  const ids = raw ? (JSON.parse(raw) as number[]) : [];
  return new Set(ids);
}

function setViewedFeedSet(viewed: Set<number>) {
  const viewedKey = 'viewer_views_feed';
  localStorage.setItem(viewedKey, JSON.stringify(Array.from(viewed)));
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
      const nextPosts = data.posts;
      setPosts((prev) => [...prev, ...nextPosts]);
      setOffset(data.nextOffset ?? offset + data.posts.length);
      setHasMore(data.hasMore ?? data.posts.length === LIMIT);

      if (typeof window !== 'undefined') {
        const token = getViewerToken();
        const viewed = getViewedFeedSet();

        await Promise.all(
          nextPosts.map(async (post) => {
            if (viewed.has(post.id)) return;
            try {
              const viewRes = await fetch(
                `${apiBase}/api/feed/${post.id}/views`,
                {
                  method: 'POST',
                  headers: {
                    'X-Viewer-Token': token,
                  },
                }
              );
              if (viewRes.ok) {
                const payload = await viewRes.json();
                setPosts((current) =>
                  current.map((item) =>
                    item.id === post.id
                      ? { ...item, views: payload.count }
                      : item
                  )
                );
                viewed.add(post.id);
              }
            } catch (viewError) {
              console.error('Failed to record view', viewError);
            }
          })
        );

        setViewedFeedSet(viewed);
      }
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
    <div className="frame border-t-0 border-b-0">
      <div>
        {posts.map((post, index) => (
          <article
            key={post.id}
            className="p-5 space-y-3 border-b border-border last:border-b-0"
          >
            <div className="flex flex-wrap items-start justify-between gap-3">
              <span className="caption">{formatDate(post.createdAt)}</span>
              <span className="text-[11px] uppercase tracking-[0.3em] font-mono text-muted-foreground">
                {post.views?.toLocaleString()} views
              </span>
            </div>
            <div
              className="feed-content text-sm leading-relaxed text-muted-foreground"
              dangerouslySetInnerHTML={{ __html: post.content }}
            />
          </article>
        ))}

        {error && (
          <div className="p-4 text-sm text-destructive">
            {error}
          </div>
        )}
      </div>

      <div ref={sentinelRef} className="h-8"></div>

      <div className="border-t border-border border-b border-border p-4 text-xs font-mono uppercase tracking-[0.25em] text-muted-foreground">
        {loading ? 'Loading' : hasMore ? 'Scroll to load more' : 'End of feed'}
      </div>
    </div>
  );
}

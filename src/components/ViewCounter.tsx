import React, { useEffect, useState } from 'react';
import { getApiBase } from '../lib/api';

interface ViewCounterProps {
    slug: string;
}

export default function ViewCounter({ slug }: ViewCounterProps) {
    const [views, setViews] = useState<number | null>(null);
    const apiBase = getApiBase();

    useEffect(() => {
        const fetchViews = async () => {
            try {
                const tokenKey = 'viewer_token';
                const viewedKey = 'viewer_views';
                let token = localStorage.getItem(tokenKey);
                if (!token) {
                    token = crypto.randomUUID();
                    localStorage.setItem(tokenKey, token);
                }

                const viewedRaw = localStorage.getItem(viewedKey);
                const viewed: string[] = viewedRaw ? JSON.parse(viewedRaw) : [];
                const hasViewed = viewed.includes(slug);

                const method = hasViewed ? 'GET' : 'POST';

                const res = await fetch(`${apiBase}/api/views/${slug}`, {
                    method: method,
                    headers: {
                        'X-Viewer-Token': token,
                    },
                });

                if (res.ok) {
                    const data = await res.json();
                    setViews(data.count);

                    if (!hasViewed) {
                        const nextViewed = [...viewed, slug];
                        localStorage.setItem(viewedKey, JSON.stringify(nextViewed));
                    }
                }
            } catch (error) {
                console.error('Failed to fetch views:', error);
            }
        };

        fetchViews();
    }, [slug]);

    if (views === null) {
        return <span>...</span>;
    }

    return (
        <span className="text-xs font-mono uppercase tracking-[0.2em] text-muted-foreground">
            {views.toLocaleString()} views
        </span>
    );
}

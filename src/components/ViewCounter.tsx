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
                const storageKey = `viewed_${slug}`;
                const hasViewed = sessionStorage.getItem(storageKey);

                const method = hasViewed ? 'GET' : 'POST';

                const res = await fetch(`${apiBase}/api/views/${slug}`, {
                    method: method,
                });

                if (res.ok) {
                    const data = await res.json();
                    setViews(data.count);

                    if (!hasViewed) {
                        sessionStorage.setItem(storageKey, 'true');
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

import React, { useEffect, useState } from 'react';

interface ViewCounterProps {
    slug: string;
}

export default function ViewCounter({ slug }: ViewCounterProps) {
    const [views, setViews] = useState<number | null>(null);

    useEffect(() => {
        const fetchViews = async () => {
            try {
                const storageKey = `viewed_${slug}`;
                const hasViewed = sessionStorage.getItem(storageKey);

                const method = hasViewed ? 'GET' : 'POST';

                const res = await fetch(`/api/views/${slug}`, {
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
        return <span className="animate-pulse">...</span>;
    }

    return (
        <span className="text-sm text-gray-500">
            {views.toLocaleString()} views
        </span>
    );
}

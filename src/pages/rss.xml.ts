import rss from "@astrojs/rss";
import { getCollection } from "astro:content";

import type { APIContext } from "astro";

export async function GET(context: APIContext) {
  const blogPosts = (await getCollection("blog"))
    .filter(post => !post.data.draft)
    .sort((a, b) => new Date(b.data.date).valueOf() - new Date(a.data.date).valueOf());

  return rss({
    title: "Furkancan Okal's Blog",
    description: "A collection of thoughts and projects by Furkancan Okal.",
    site: context.site?.toString() ?? "",
    items: blogPosts.map((post) => ({
      title: post.data.title,
      description: post.data.description,
      pubDate: post.data.date,
      link: `/blog/${post.slug}/`,
    })),
  });
}

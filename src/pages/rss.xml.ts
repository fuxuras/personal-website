import rss from "@astrojs/rss";
import { getCollection } from "astro:content";

type Context = {
  site: string
}

export async function GET(context: Context) {
  const blogPosts = (await getCollection("blog"))
    .filter(post => !post.data.draft)
    .sort((a, b) => new Date(b.data.date).valueOf() - new Date(a.data.date).valueOf());

  return rss({
    title: "Fuxuras's Blog", // Generic title
    description: "A collection of thoughts and projects by Fuxuras.", // Generic description
    site: context.site,
    items: blogPosts.map((post) => ({
      title: post.data.title,
      description: post.data.description,
      pubDate: post.data.date,
      link: `/blog/${post.slug}/`,
    })),
  });
}

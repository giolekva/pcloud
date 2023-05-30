export const ssr = false;

export type AppData = {
  slug: string;
  schema: Record<string, unknown>;
  config: Record<string, unknown>;
};

export async function load({ params, fetch }): Promise<AppData> {
  const resp =  await fetch(`/api/app/${params.slug}`);
  const ret = await resp.json();
  return {
    slug: params.slug,
    schema: JSON.parse(ret.schema),
    config: ret.config,
  };
}

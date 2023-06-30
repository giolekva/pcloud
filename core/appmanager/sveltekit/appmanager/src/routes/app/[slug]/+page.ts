export const ssr = false;

export type AppData = {
  name: string;
  icon: string;
  slug: string;
  schema: Record<string, unknown>;
  instances: unknown[];
};

export async function load({ params, fetch }): Promise<AppData> {
  const resp =  await fetch(`/api/app/${params.slug}`);
  const ret = await resp.json();
  console.log(ret);
  return {
    name: ret.name,
    icon: ret.icon,
    slug: params.slug,
    schema: JSON.parse(ret.schema),
    instances: ret.instances,
  };
}

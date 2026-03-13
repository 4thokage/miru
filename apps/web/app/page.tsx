import Search from "@/components/Search";
import Link from "next/link";

export default function Home() {
  return (
    <div className="min-h-screen bg-zinc-50 dark:bg-black">
      <header className="py-8 px-4 text-center border-b border-zinc-200 dark:border-zinc-800">
        <h1 className="text-3xl font-bold text-zinc-900 dark:text-zinc-50">
          Miru
        </h1>
        <p className="text-zinc-600 dark:text-zinc-400 mt-2">
          Discover and read manga
        </p>
        <div className="flex justify-center gap-4 mt-4">
          <Link href="/" className="text-blue-600 dark:text-blue-400 font-medium">
            Manga
          </Link>
          <Link href="/anime" className="text-zinc-600 dark:text-zinc-400 hover:text-blue-600 dark:hover:text-blue-400">
            Anime
          </Link>
        </div>
      </header>
      <main>
        <Search />
      </main>
    </div>
  );
}

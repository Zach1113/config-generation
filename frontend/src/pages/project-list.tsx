import { useState } from "react"
import { useProjects } from "@/hooks/use-projects"
import { ProjectCard } from "@/components/projects/project-card"
import { CreateProjectDialog } from "@/components/projects/create-project-dialog"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { FolderKanban, Search, SearchX } from "lucide-react"

export default function ProjectListPage() {
  const { data, isLoading, error } = useProjects()
  const [search, setSearch] = useState("")

  const filtered = data?.items.filter((p) =>
    p.name.toLowerCase().includes(search.toLowerCase()),
  ) ?? []

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold">Projects</h1>
          <p className="text-sm text-muted-foreground">
            Manage configuration workspaces for each service or application.
          </p>
        </div>
        <CreateProjectDialog />
      </div>

      <div className="flex flex-col gap-3 rounded-lg border bg-card p-3 shadow-xs sm:flex-row sm:items-center sm:justify-between">
        <div className="relative w-full sm:max-w-md">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder="Search projects..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <p className="text-sm text-muted-foreground">
          {data?.count ?? 0} {data?.count === 1 ? "project" : "projects"}
        </p>
      </div>

      {isLoading && (
        <p className="text-muted-foreground">Loading projects...</p>
      )}

      {error && (
        <p className="text-destructive">
          Failed to load projects: {(error as Error).message}
        </p>
      )}

      {!isLoading && filtered.length === 0 ? (
        <div className="flex min-h-72 items-center justify-center rounded-lg border border-dashed bg-card/40 p-8 text-center">
          <div className="mx-auto flex max-w-sm flex-col items-center gap-3">
            <div className="flex size-11 items-center justify-center rounded-lg border bg-background text-muted-foreground">
              {search ? (
                <SearchX className="h-5 w-5" />
              ) : (
                <FolderKanban className="h-5 w-5" />
              )}
            </div>
            <div className="space-y-1">
              <h2 className="text-base font-semibold">
                {search ? "No matching projects" : "No projects yet"}
              </h2>
              <p className="text-sm text-muted-foreground">
                {search
                  ? "Try a different search term or clear the search field."
                  : "Create your first project to start organizing templates and environments."}
              </p>
            </div>
            {search && (
              <Button variant="outline" onClick={() => setSearch("")}>
                Clear search
              </Button>
            )}
          </div>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {filtered.map((project) => (
            <ProjectCard key={project.id} project={project} />
          ))}
        </div>
      )}
    </div>
  )
}

import { NavLink } from 'react-router-dom'

const tab = ({ isActive }: { isActive: boolean }) =>
  `border-b-2 px-1 pb-2 text-sm ${
    isActive ? 'border-indigo-400 text-white' : 'border-transparent text-slate-400 hover:text-white'
  }`

/** Tabs for the pages of one workspace. */
export function WorkspaceNav({ slug }: { slug: string }) {
  return (
    <nav aria-label="Workspace pages" className="mt-6 flex gap-6 border-b border-slate-800">
      <NavLink to={`/w/${slug}`} end className={tab}>
        Overview
      </NavLink>
      <NavLink to={`/w/${slug}/projects`} className={tab}>
        Projects
      </NavLink>
      <NavLink to={`/w/${slug}/members`} className={tab}>
        Members
      </NavLink>
    </nav>
  )
}
import { useEffect } from 'react'
import { Outlet, NavLink, useLocation } from 'react-router-dom'
import { useQueueCount } from '../features/documents/hooks/useQueueCount'
import { useAppNav } from '../lib/appNav'

function IconLetters({ className }: { className?: string }) {
  return (
    <svg className={className} width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
      <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
      <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z" />
      <path d="M8 7h8M8 11h8M8 15h5" />
    </svg>
  )
}

function IconFolders({ className }: { className?: string }) {
  return (
    <svg className={className} width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
      <path d="M3 8V6a1 1 0 0 1 1-1h4l2 2h6a1 1 0 0 1 1 1v1" />
      <path d="M3 10a1 1 0 0 1 1-1h16a1 1 0 0 1 1 1v8a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1v-8z" />
    </svg>
  )
}

function IconPlus({ className }: { className?: string }) {
  return (
    <svg className={className} width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.25" strokeLinecap="round" aria-hidden>
      <path d="M12 5v14M5 12h14" />
    </svg>
  )
}

function IconGear({ className }: { className?: string }) {
  return (
    <svg className={className} width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
      <circle cx="12" cy="12" r="3" />
      <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" />
    </svg>
  )
}

/** Desktop sidebar nav link */
function SideNavLink({
  to,
  end,
  children,
  badge,
}: {
  to: string
  end?: boolean
  children: React.ReactNode
  badge?: number
}) {
  return (
    <NavLink
      to={to}
      end={end}
      aria-label={badge && badge > 0 ? `${String(children)}, ${badge} in queue` : undefined}
      className={({ isActive }) =>
        `flex items-center gap-2 px-3 py-2.5 rounded-btn text-sm font-medium transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/40 focus-visible:ring-offset-2 focus-visible:ring-offset-card ${
          isActive ? 'bg-accent/10 text-accent' : 'text-gray-800 hover:bg-surface'
        }`
      }
    >
      <span className="flex-1 min-w-0 truncate">{children}</span>
      {badge && badge > 0 ? (
        <span
          aria-hidden
          className="tabular-nums text-xs font-semibold bg-accent/15 text-accent rounded-btn px-1.5 py-0.5"
        >
          {badge}
        </span>
      ) : null}
    </NavLink>
  )
}

/** Compact page label for the mobile top strip (right side). */
function mobileTopTitle(pathname: string): string | null {
  if (pathname === '/') return 'My letters'
  if (pathname === '/explore/no-date') return 'No date'
  if (pathname.startsWith('/explore')) {
    // Folder pages keep their own name visible while the list scrolls.
    const folder = pathname.slice('/explore'.length).replace(/^\/|\/$/g, '')
    return folder ? decodeURIComponent(folder) : 'Explore'
  }
  if (pathname.startsWith('/settings')) return 'Settings'
  if (pathname.startsWith('/add')) return 'Scan letters'
  return null
}

export default function Layout() {
  const location = useLocation()
  const { appPath, stripPrefix } = useAppNav()
  const path = stripPrefix(location.pathname)
  const isCamera = path === '/add/camera'
  const topTitle = mobileTopTitle(path)
  const { total: queueTotal } = useQueueCount()
  const queueBadge = queueTotal > 0 ? queueTotal : 0
  const lettersAria =
    queueBadge > 0 ? `My letters, ${queueBadge} in queue` : 'My letters'

  /** Lock document scroll on camera — mobile Safari otherwise rubber-bands / scrolls past 100vh. */
  useEffect(() => {
    if (!isCamera) return
    const html = document.documentElement
    const body = document.body
    const root = document.getElementById('root')
    const scrollY = window.scrollY

    const prev = {
      htmlOverflow: html.style.overflow,
      bodyOverflow: body.style.overflow,
      htmlHeight: html.style.height,
      bodyHeight: body.style.height,
      bodyPosition: body.style.position,
      bodyWidth: body.style.width,
      bodyTop: body.style.top,
      rootOverflow: root?.style.overflow ?? '',
      rootHeight: root?.style.height ?? '',
      rootMaxH: root?.style.maxHeight ?? '',
    }

    html.style.overflow = 'hidden'
    body.style.overflow = 'hidden'
    html.style.height = '100%'
    body.style.height = '100%'
    html.style.overscrollBehavior = 'none'
    body.style.overscrollBehavior = 'none'
    body.style.position = 'fixed'
    body.style.width = '100%'
    body.style.top = `-${scrollY}px`

    if (root) {
      root.style.overflow = 'hidden'
      root.style.height = '100%'
      root.style.maxHeight = '100%'
    }

    return () => {
      html.style.overflow = prev.htmlOverflow
      body.style.overflow = prev.bodyOverflow
      html.style.height = prev.htmlHeight
      body.style.height = prev.bodyHeight
      html.style.removeProperty('overscroll-behavior')
      body.style.removeProperty('overscroll-behavior')
      body.style.position = prev.bodyPosition
      body.style.width = prev.bodyWidth
      body.style.top = prev.bodyTop
      if (root) {
        root.style.overflow = prev.rootOverflow
        root.style.height = prev.rootHeight
        root.style.maxHeight = prev.rootMaxH
      }
      window.scrollTo(0, scrollY)
    }
  }, [isCamera])

  const mainBottomPad = 'pb-[calc(4rem+env(safe-area-inset-bottom,0px))] md:pb-0'
  const mainClass = isCamera
    ? 'flex-1 min-h-0 flex flex-col overflow-hidden relative'
    : `flex-1 min-h-0 flex flex-col overflow-hidden pt-12 md:pt-0 ${mainBottomPad}`

  const shellClass = isCamera
    ? 'min-h-0 h-[100dvh] max-h-[100dvh] bg-surface flex overflow-hidden overscroll-none'
    : 'h-screen bg-surface flex overflow-hidden'

  return (
    <div className={shellClass}>
      {/* Desktop: vertical sidebar; mobile: hidden — bottom tabs; camera route: hidden */}
      <aside
        className={
          isCamera
            ? 'hidden'
            : 'hidden md:flex md:w-56 md:flex-shrink-0 md:flex-col border-r border-border bg-card shadow-card z-10'
        }
      >
        <div className="h-16 px-4 border-b border-border flex items-center gap-3 flex-shrink-0">
          <div className="w-8 h-8 rounded-btn bg-accent flex items-center justify-center flex-shrink-0 shadow-sm" aria-hidden>
            <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
              <path d="M4 3.5A1.5 1.5 0 0 1 5.5 2H10l4 4v8.5a1.5 1.5 0 0 1-1.5 1.5h-7A1.5 1.5 0 0 1 4 14.5V3.5z" fill="white" fillOpacity="0.9" />
              <path d="M10 2v4h4" fill="white" fillOpacity="0.4" />
              <path d="M6.5 9h5M6.5 11.5h3.5" stroke="white" strokeWidth="1.2" strokeLinecap="round" />
            </svg>
          </div>
          <span className="font-semibold text-gray-900 text-sm tracking-tight">Sonix</span>
        </div>
        <nav className="flex-1 p-3 space-y-0.5 overflow-y-auto" aria-label="Main">
          <SideNavLink to={appPath('/')} end badge={queueBadge}>
            My letters
          </SideNavLink>
          <SideNavLink to={appPath('/explore')}>Explore</SideNavLink>
          <SideNavLink to={appPath('/add')}>Scan letters</SideNavLink>
          <SideNavLink to={appPath('/settings')}>Settings</SideNavLink>
        </nav>
      </aside>

      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        <header
          className={
            isCamera
              ? 'hidden'
              : 'md:hidden fixed top-0 left-0 right-0 z-30 h-12 flex items-center gap-2 px-4 border-b border-border bg-card/90 backdrop-blur-md shadow-nav'
          }
        >
          <div className="w-7 h-7 rounded-btn bg-accent flex items-center justify-center flex-shrink-0 shadow-sm" aria-hidden>
            <svg width="16" height="16" viewBox="0 0 18 18" fill="none">
              <path d="M4 3.5A1.5 1.5 0 0 1 5.5 2H10l4 4v8.5a1.5 1.5 0 0 1-1.5 1.5h-7A1.5 1.5 0 0 1 4 14.5V3.5z" fill="white" fillOpacity="0.9" />
              <path d="M10 2v4h4" fill="white" fillOpacity="0.4" />
              <path d="M6.5 9h5M6.5 11.5h3.5" stroke="white" strokeWidth="1.2" strokeLinecap="round" />
            </svg>
          </div>
          <span className="font-semibold text-gray-900 text-sm tracking-tight">Sonix</span>
          {topTitle ? (
            <span className="ml-auto text-sm font-medium text-gray-800 truncate max-w-[50%] text-right">
              {topTitle}
            </span>
          ) : null}
        </header>

        <main className={mainClass}>
          <Outlet />
        </main>

        <nav
          className={
            isCamera
              ? 'hidden'
              : 'fixed bottom-0 left-0 right-0 z-20 md:hidden border-t border-border bg-card/95 backdrop-blur-md flex items-stretch justify-around h-16 pb-[env(safe-area-inset-bottom)] shadow-[0_-4px_12px_-2px_rgb(0_0_0/0.06)]'
          }
          aria-label="Primary"
        >
          <NavLink
            to={appPath('/')}
            end
            aria-label={lettersAria}
            className={({ isActive }) =>
              `relative flex flex-1 flex-col items-center justify-center gap-0.5 min-w-0 py-1 rounded-btn transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-accent/35 ${
                isActive ? 'text-accent' : 'text-muted hover:text-gray-800'
              }`
            }
          >
            <span className="relative">
              <IconLetters className="opacity-90" />
              {queueBadge > 0 ? (
                <span
                  aria-hidden
                  className="absolute -top-1 -right-2 min-w-[1.1rem] h-[1.1rem] px-0.5 rounded-full bg-accent text-white text-[10px] font-semibold leading-[1.1rem] text-center tabular-nums"
                >
                  {queueBadge > 99 ? '99+' : queueBadge}
                </span>
              ) : null}
            </span>
            <span className="text-xs font-medium leading-none truncate max-w-full px-1">My letters</span>
          </NavLink>
          <NavLink
            to={appPath('/explore')}
            className={({ isActive }) =>
              `flex flex-1 flex-col items-center justify-center gap-0.5 min-w-0 py-1 rounded-btn transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-accent/35 ${
                isActive ? 'text-accent' : 'text-muted hover:text-gray-800'
              }`
            }
          >
            <IconFolders className="opacity-90" />
            <span className="text-xs font-medium leading-none truncate max-w-full px-1">Explore</span>
          </NavLink>
          <NavLink
            to={appPath('/add')}
            className={({ isActive }) =>
              `flex flex-1 flex-col items-center justify-center gap-0.5 min-w-0 py-1 rounded-btn transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-accent/35 ${
                isActive ? 'text-accent' : 'text-muted hover:text-gray-800'
              }`
            }
          >
            <span
              className={`w-11 h-11 rounded-full flex items-center justify-center shadow-card ${
                path.startsWith('/add') ? 'bg-accent text-white' : 'bg-accent/15 text-accent'
              }`}
            >
              <IconPlus className={path.startsWith('/add') ? 'text-white' : 'text-accent'} />
            </span>
            {/* Short label so four tabs fit without truncation at 360 CSS px. */}
            <span className="text-xs font-medium leading-none text-gray-900">Scan</span>
          </NavLink>
          <NavLink
            to={appPath('/settings')}
            className={({ isActive }) =>
              `flex flex-1 flex-col items-center justify-center gap-0.5 min-w-0 py-1 rounded-btn transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-accent/35 ${
                isActive ? 'text-accent' : 'text-muted hover:text-gray-800'
              }`
            }
          >
            <IconGear className="opacity-90" />
            <span className="text-xs font-medium leading-none truncate max-w-full px-1">Settings</span>
          </NavLink>
        </nav>
      </div>
    </div>
  )
}

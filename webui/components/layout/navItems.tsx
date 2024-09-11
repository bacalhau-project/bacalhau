import { Home, Briefcase, Network } from 'lucide-react'
import React from 'react'

export interface NavLinkProps {
  href: string
  label: string
  icon: React.ComponentType<React.SVGProps<SVGSVGElement>>
  badge?: React.ReactNode
}

export const navLinkItems: NavLinkProps[] = [
  {
    href: '/',
    icon: Home,
    label: 'Dashboard',
  },
  {
    href: '/jobs',
    icon: Briefcase,
    label: 'Jobs',
  },
  {
    href: '/nodes',
    icon: Network,
    label: 'Nodes',
  },
]

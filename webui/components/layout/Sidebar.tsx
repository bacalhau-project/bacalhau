'use client'

import React from 'react'
import Link from 'next/link'
import { Badge } from '@/components/ui/badge'
import { BacalhauLogo } from './BacalhauLogo'
import { NavLinkProps, navLinkItems } from './navItems'

export function NavLink({ href, icon: Icon, label, badge }: NavLinkProps) {
  const handleClick = () => {
    // Dispatch a custom event when any nav link is clicked
    window.dispatchEvent(
      new CustomEvent('refreshContent', {
        detail: { page: label.toLowerCase() },
      })
    )
  }

  return (
    <Link
      href={href}
      className="flex items-center gap-3 rounded-lg px-4 py-3 text-sidebar-text transition-all hover:bg-sidebar-hover hover:text-white"
      onClick={handleClick}
    >
      <Icon className="h-5 w-5" />
      <span className="flex-grow">{label}</span>
      {badge && (
        <Badge className="ml-auto flex h-6 w-6 shrink-0 items-center justify-center rounded-full">
          {badge}
        </Badge>
      )}
    </Link>
  )
}

export function InnerSidebar() {
  return (
    <div className="flex h-full flex-col gap-4">
      <div className="flex h-14 items-center px-4 lg:h-[60px] lg:px-6">
        <Link
          href="/"
          className="flex items-center gap-2 font-semibold text-sidebar-text"
        >
          <BacalhauLogo
            colorOption={'white'}
            className="w-full md:w-[180px] lg:w-[230px]"
          />
        </Link>
      </div>
      <div className="flex-1 overflow-y-auto">
        <nav className="grid items-start gap-1 px-4 text-sm font-medium">
          {navLinkItems.map((item) => (
            <NavLink key={item.href} {...item} />
          ))}
        </nav>
      </div>
    </div>
  )
}

export function Sidebar() {
  return (
    <div className="hidden md:block fixed top-0 left-0 h-full w-[220px] lg:w-[280px] border-r border-sidebar-border bg-sidebar-bg overflow-y-auto z-40">
      <InnerSidebar />
    </div>
  )
}

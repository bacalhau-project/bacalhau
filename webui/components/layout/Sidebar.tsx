import React from 'react'
import Link from 'next/link'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { BacalhauLogo } from './BacalhauLogo'
import { NavLinkProps, navLinkItems } from './navItems'

function NavLink({ href, icon: Icon, label, badge }: NavLinkProps) {
  return (
    <Link
      href={href}
      className="flex items-center gap-3 rounded-lg px-4 py-3 text-sidebar-text transition-all hover:bg-sidebar-hover hover:text-white"
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

export function Sidebar() {
  return (
    <div className="hidden border-r border-sidebar-border bg-sidebar-bg md:block">
      <div className="flex h-full max-h-screen flex-col gap-4">
        <div className="flex h-14 items-center px-4 lg:h-[60px] lg:px-6">
          <Link
            href="/"
            className="flex items-center gap-2 font-semibold text-sidebar-text"
          >
            <BacalhauLogo colorOption={'white'} />
          </Link>
        </div>
        <div className="flex-1 overflow-auto">
          <nav className="grid items-start gap-1 px-4 text-sm font-medium">
            {navLinkItems.map((item) => (
              <NavLink key={item.href} {...item} />
            ))}
          </nav>
        </div>
        <div className="mt-auto p-4">
          <Card className="bg-sidebar-bg border-sidebar-border text-sidebar-text">
            <CardHeader className="p-4">
              <CardTitle className="text-white">Upgrade to Pro</CardTitle>
              <CardDescription className="text-gray-400">
                Unlock all features and get unlimited access to our support
                team.
              </CardDescription>
            </CardHeader>
            <CardContent className="p-4 pt-0">
              <Button
                size="sm"
                className="w-full bg-button-primary text-white hover:bg-button-primaryHover"
              >
                Upgrade
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}

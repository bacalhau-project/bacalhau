import React from 'react';
import { HelpCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { MobileNav } from './MobileNav';
import { ThemeToggle } from "./ThemeToggle";

export function Header() {
  return (
      <header className="flex h-14 items-center gap-4 border-b bg-muted/40 px-4 lg:h-[60px] lg:px-6">
        <MobileNav />
        <div className="w-full flex-1"></div> {/* Empty space where the search bar used to be */}

        <ThemeToggle />
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" className="w-9 px-0">
              <HelpCircle className="h-5 w-5" />
              <span className="sr-only">Help menu</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem>
              <a href="https://docs.bacalhau.org" target="_blank" rel="noopener noreferrer">
                Documentation
              </a>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </header>
  );
}

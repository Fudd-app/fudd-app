import React from "react";
import Link from "next/link";
import Logo from "@/components/brand/Logo";

export default function RegistroBrand() {
  return (
    <Link href="/" className="auth-brand">
      <span><Logo className="w-full h-full" /></span>zelko
    </Link>
  );
}

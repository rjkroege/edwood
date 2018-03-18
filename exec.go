package main

import (
//	"path/filepath"

//	"9fans.net/go/plan9/client"
)

func execute(t *Text, q0, q1 int, external bool, argt *Text) {
	_, _, _, _, _ = t, q0, q1, external, argt
	Unimpl()
}

func cut(et *Text, t *Text, _0 *Text, dosnarf bool, docut bool, _2 []rune, _3 int) {
	Unimpl()
}

func paste(et *Text, t *Text, _0 *Text, selectall bool, tobody bool, _2 []rune, _3 int) {
	Unimpl()
}

func get(et * Text, t * Text, argt * Text, flag1  bool, _0  bool, arg []rune, narg  int){
	Unimpl()
}
func put(et * Text, _0 * Text, argt * Text, _1  bool, _2  bool, arg []rune, narg  int){
	Unimpl()
}

func undo(et * Text, _0 * Text, _1 * Text, flag1  bool, _2  bool, _3 []rune, _4  int) {
	Unimpl()
}

func run(win *Window, s []rune, rdir []rune, newns bool, argaddr []rune, xarg []rune, iseditcmd bool) {
	var (
		c    *Command
		cpid chan int
	)

	if len(s) == 0 {
		return
	}

	c = &Command{}
	cpid = make(chan int)
	go runproc(win, s, rdir, newns, argaddr, xarg, c, cpid, iseditcmd)

	// Wait for runproc to signal
	var pid uint64
	for {
		pid := <-cpid
		if pid != ^0 {
			break
		}
	}

	/* mustn't block here because must be ready to answer mount() call in run() */

	if pid != 0 {
		ccommand <- c
	} else {
		if c.iseditcommand {
			cedit <- 0
		}
	}
	//arg = make([]interface{},2)
	//arg[0] = c;
	//arg[1] = cpid;
	//threadcreate(runwaittask, arg, STACK);
}

func runproc(win *Window, s []rune, rdir []rune, newns bool, argaddr []rune, xarg []rune, c *Command, cpid chan int, iseditcmd bool) {
	Unimpl()
} /*
	var (
	e, t, name, filename, dir, news string
	av []string
	r rune
	incl [][]rune
	ac, w, inarg, i, n, fd, nincl, winid int
	isfd[3] int
	pipechar int
	buf [512]byte
	ret int
	//static void *parg[2];
	rcarg[4]string
	argv []string
	fs *Fsys
	shell string
	)

	t = runeTrimLeft(s, []rune(" \t\n"))
	name := filepath.Base(string(t)) + " "
	c.name = []rune(name)
	// t is the full path, trimmed of left whitespace.
	pipechar = 0;

	if t[0]=='<' || t[0]=='|' || t[0]=='>' {
		pipechar = t[0]
		t = t[1:]
	}
	c.iseditcmd = iseditcmd;
	c.text = s;
	if newns {
		nincl = 0;
		incl = nil;
		if win {
			filename = string(win.body.file.name);
			if len(incl) > 0 {
				incl = male([]string, len(incl)) // TODO(flux): I don't know why we duplicate this struct.
				for i, inc := range win.incl {
					incl[i] = inc
				}
			}
			winid = win.id;
		}else{
			filename = nil;
			winid = 0;
			if activewin {
				winid = activewin.id;
			}
		}
	// 	rfork(RFNAMEG|RFENVG|RFFDG|RFNOTEG); TODO(flux): I'm sure these settings are important

		sprint(buf, "%d", winid);
		putenv("winid", buf);

		if filename {
			putenv("%", filename);
			putenv("samfile", filename);
			free(filename);
		}
		c.md = fsysmount(rdir, incl);
		if c.md == nil {
			fprint(2, "child: can't allocate mntdir: %r\n");
			threadexits("fsysmount");
		}
		sprint(buf, "%d", c.md.id);
		fs, err := plan9.NsMount("acme", buf)
		if err == nil {
			fmt.Fprintf(os.Stderr, "child: can't mount acme: %r\n", err);
			fsysdelid(c.md);
			c.md = nil;
			threadexits("nsmount");
		}
		if winid>0 && (pipechar=='|' || pipechar=='>') {
			sprint(buf, "%d/rdsel", winid);
			sfd[0] = fsopenfd(fs, buf, OREAD);
		}else
			sfd[0] = open("/dev/null", OREAD);
		if (winid>0 || iseditcmd) && (pipechar=='|' || pipechar=='<') {
			if iseditcmd {
				if winid > 0
					sprint(buf, "%d/editout", winid);
				else
					sprint(buf, "editout");
			}else
				sprint(buf, "%d/wrsel", winid);
			sfd[1] = fsopenfd(fs, buf, OWRITE);
			sfd[2] = fsopenfd(fs, "cons", OWRITE);
		}else{
			sfd[1] = fsopenfd(fs, "cons", OWRITE);
			sfd[2] = sfd[1];
		}
		fsunmount(fs);
	}else{
		rfork(RFFDG|RFNOTEG);
		fsysclose();
		sfd[0] = open("/dev/null", OREAD);
		sfd[1] = open("/dev/null", OWRITE);
		sfd[2] = dup(erroutfd, -1);
	}
	if win
		winclose(win);

	if argaddr
		putenv("acmeaddr", argaddr);
	if acmeshell != nil
		goto Hard;
	if strlen(t) > sizeof buf-10 	// may need to print into stack
		goto Hard;
	inarg = false;
	for e=t; *e; e+=w {
		w = chartorune(&r, e);
		if r==' ' || r=='\t'
			continue;
		if r < ' '
			goto Hard;
		if utfrune("#;&|^$=`'{}()<>[]*?^~`/", r)
			goto Hard;
		inarg = true;
	}
	if !inarg
		goto Fail;

	ac = 0;
	av = nil;
	inarg = false;
	for e=t; *e; e+=w {
		w = chartorune(&r, e);
		if r==' ' || r=='\t' {
			inarg = false;
			*e = 0;
			continue;
		}
		if !inarg {
			inarg = true;
			av = realloc(av, (ac+1)*sizeof(char**));
			av[ac++] = e;
		}
	}
	av = realloc(av, (ac+2)*sizeof(char**));
	av[ac++] = arg;
	av[ac] = nil;
	c.av = av;

	dir = nil;
	if rdir != nil
		dir = runetobyte(rdir, ndir);
	ret = threadspawnd(sfd, av[0], av, dir);
	free(dir);
	if ret >= 0 {
		if cpid
			sendul(cpid, ret);
		threadexits("");
	}
// libthread uses execvp so no need to do this
//#if 0
//	e = av[0];
//	if e[0]=='/' || (e[0]=='.' && e[1]=='/')
//		goto Fail;
//	if cputype {
//		sprint(buf, "%s/%s", cputype, av[0]);
//		procexec(cpid, sfd, buf, av);
//	}
//	sprint(buf, "/bin/%s", av[0]);
//	procexec(cpid, sfd, buf, av);
//#endif
	goto Fail;

Hard:
	 //* ugly: set path = (. $cputype /bin)
	 //* should honor $path if unusual.
	if cputype {
		n = 0;
		memmove(buf+n, ".", 2);
		n += 2;
		i = strlen(cputype)+1;
		memmove(buf+n, cputype, i);
		n += i;
		memmove(buf+n, "/bin", 5);
		n += 5;
		fd = create("/env/path", OWRITE, 0666);
		write(fd, buf, n);
		close(fd);
	}

	if arg {
		news = emalloc(strlen(t) + 1 + 1 + strlen(arg) + 1 + 1);
		if news {
			sprint(news, "%s '%s'", t, arg);	// BUG: what if quote in arg?
			free(s);
			t = news;
			c.text = news;
		}
	}
	dir = nil;
	if rdir != nil
		dir = runetobyte(rdir, ndir);
	shell = acmeshell;
	if shell == nil
		shell = "rc";
	rcarg[0] = shell;
	rcarg[1] = "-c";
	rcarg[2] = t;
	rcarg[3] = nil;
	ret = threadspawnd(sfd, rcarg[0], rcarg, dir);
	free(dir);
	if ret >= 0 {
		if cpid
			sendul(cpid, ret);
		threadexits(nil);
	}
	warning(nil, "exec %s: %r\n", shell);

   Fail:
	// threadexec hasn't happened, so send a zero
	close(sfd[0]);
	close(sfd[1]);
	if sfd[2] != sfd[1]
		close(sfd[2]);
	sendul(cpid, 0);
	threadexits(nil);
}
*/

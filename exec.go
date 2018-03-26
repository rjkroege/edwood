package main

import (
	"fmt"
	"regexp"
	"strings"
//	"path/filepath"

//	"9fans.net/go/plan9/client"
)


type Exectab struct {
	name string
	fn func(t0, t1, t2 *Text, b0, b1 bool, arg string)
	mark bool
	flag1 bool
	flag2 bool
}

var exectab =  []Exectab{
//	{ "Abort",		doabort,	false,	true /*unused*/,		true /*unused*/,		},
//	{ "Cut",		cut,		true,	true,	true	},
	{ "Del",		del,		false,	false,	true /*unused*/		},
//	{ "Delcol",		delcol,	false,	true /*unused*/,		true /*unused*/		},
	{ "Delete",		del,		false,	true,	true /*unused*/		},
//	{ "Dump",		dump,	false,	true,	true /*unused*/		},
//	{ "Edit",		edit,		false,	true /*unused*/,		true /*unused*/		},
	{ "Exit",		xexit,	false,	true /*unused*/,		true /*unused*/		},
//	{ "Font",		fontx,	false,	true /*unused*/,		true /*unused*/		},
//	{ "Get",		get,		false,	true,	true /*unused*/		},
//	{ "ID",		id,		false,	true /*unused*/,		true /*unused*/		},
//	{ "Incl",		incl,		false,	true /*unused*/,		true /*unused*/		},
//	{ "Indent",		indent,	false,	true /*unused*/,		true /*unused*/		},
//	{ "Kill",		xkill,		false,	true /*unused*/,		true /*unused*/		},
//	{ "Load",		dump,	false,	false,	true /*unused*/		},
//	{ "Local",		local,	false,	true /*unused*/,		true /*unused*/		},
//	{ "Look",		look,		false,	true /*unused*/,		true /*unused*/		},
//	{ "New",		new,		false,	true /*unused*/,		true /*unused*/		},
//	{ "Newcol",	newcol,	false,	true /*unused*/,		true /*unused*/		},
//	{ "Paste",		paste,	true,	true,	true /*unused*/		},
//	{ "Put",		put,		false,	true /*unused*/,		true /*unused*/		},
//	{ "Putall",		putall,	false,	true /*unused*/,		true /*unused*/		},
//	{ "Redo",		undo,	false,	false,	true /*unused*/		},
//	{ "Send",		sendx,	true,	true /*unused*/,		true /*unused*/		},
//	{ "Snarf",		cut,		false,	true,	false	},
//	{ "Sort",		sort,		false,	true /*unused*/,		true /*unused*/		},
//	{ "Tab",		tab,		false,	true /*unused*/,		true /*unused*/		},
//	{ "Undo",		undo,	false,	true,	true /*unused*/		},
//	{ "Zerox",		zeroxx,	false,	true /*unused*/,		true /*unused*/		},
}

var wsre = regexp.MustCompile("[ \t\n]+")

func lookup(r string) *Exectab {
fmt.Println("lookup", r)
	r = wsre.ReplaceAllString(r, " ")
	r = strings.TrimLeft(r, " ")
	r = strings.SplitN(r, " ", 1)[0]
	for _, e := range exectab {
		if e.name == r {
			return &e
		}
	}
	return nil
}

func isexecc(c rune) bool {
	if isfilec(c) { return true }
	return c == '<' || c == '|' || c == '>'
}

func printarg(argt * Text, q0  int, q1  int) string {
	if(argt.what!=Body || argt.file.name=="") {
		return "";
	}
	if(q0 == q1) {
		return fmt.Sprintf("%s:#%d", argt.file.name, q0);
	} else {
		return fmt.Sprintf("%s:#%d,#%d", argt.file.name, q0, q1);
	}
}


func getarg(argt * Text, doaddr  bool, dofile  bool) (string, string) {
	if(argt == nil) {
		return "", "";
	}
	a := ""
	var e Expand
	argt.Commit(true);
	var ok bool
	if e, ok = expand(argt, argt.q0, argt.q1); ok {
		if(len(e.name)>0 && dofile){
			if(doaddr) {
				a = printarg(argt, e.q0, e.q1);
			}
			return e.name, a;
		}
	}else{
		e.q0 = argt.q0;
		e.q1 = argt.q1;
	}
	n := e.q1 - e.q0;
	r := argt.file.b.Read(e.q0, n);
	if(doaddr) {
		a = printarg(argt, e.q0, e.q1);
	}
	return string(r), a;
}


func execute( t * Text, aq0  int, aq1  int, external  bool, argt * Text) {
Untested()
var (
	q0, q1 int
	r []rune
	n, f int
	dir string
)

	q0 = aq0;
	q1 = aq1;
	if(q1 == q0){	// expand to find word (actually file name) 
		fmt.Println("q1 == q0")
		// if in selection, choose selection 
		if(t.inSelection(q0)){
			q0 = t.q0;
			q1 = t.q1;
		fmt.Println("selection chosen")
		}else{
			for(q1<t.file.b.nc()){
				c:=t.ReadC(q1)
				 if isexecc(c) && c!=':' {
					q1++;
				} else { break }
			}
			for q0>0 {
				c:=t.ReadC(q0-1)
				if isexecc(c) && c!=':' {
					q0--;
				} else {
					break
				}
			}
		fmt.Println("expanded selection")
			if(q1 == q0) {
		fmt.Println("selection chosen")
				return;
			}
		}
	}
	r = t.file.b.Read(q0, q1-q0)
	e := lookup(string(r));
	if(!external && t.w!=nil && t.w.nopen[QWevent]>0){
		f = 0;
		if(e != nil) {
			f |= 1;
		}
		if(q0!=aq0 || q1!=aq1){
			r = t.file.b.Read(aq0, aq1-aq0);
			f |= 2;
		}
		aa, a := getarg(argt, true, true);
		if(a != ""){	
			if(len(a) > EVENTSIZE){	// too big; too bad 
				warning(nil, "argument string too long\n");
				return;
			}
			f |= 8;
		}
		c := 'x';
		if(t.what == Body) {
			c = 'X';
		}
		n = aq1-aq0;
		if(n <= EVENTSIZE) {
			t.w.Event("%c%d %d %d %d %s\n", c, aq0, aq1, f, n, r);
		} else {
			t.w.Event("%c%d %d %d 0 \n", c, aq0, aq1, f, n);
		}
		if(q0!=aq0 || q1!=aq1){
			n = q1-q0;
			r := t.file.b.Read(q0, n);
			if(n <= EVENTSIZE) {
				t.w.Event("%c%d %d 0 %d %s\n", c, q0, q1, n, r);
			} else {
				t.w.Event("%c%d %d 0 0 \n", c, q0, q1, n);
			}
		}
		if(a!=""){
			t.w.Event("%c0 0 0 %d %s\n", c, len(a), a);
			if(aa != "") {
				t.w.Event("%c0 0 0 %d %s\n", c, len(aa), aa);
			} else {
				t.w.Event("%c0 0 0 0 \n", c);
			}
		}
		return;
	}
	if(e!=nil){
		if(e.mark && seltext!=nil) && seltext.what == Body{
			seq++;
			seltext.w.body.file.Mark();
		}
		s := wsre.ReplaceAllString(string(r), " ")
		s = strings.TrimLeft(s, " ")
		words := strings.SplitN(s, " ", 2)
		if len(words) == 1 {
			words = append(words, "")
		}
		e.fn(t, seltext, argt, e.flag1, e.flag2, words[1]);
		return;
	}

	b := r
	dir = t.DirName();
	if(dir=="."){	// sigh 
		dir= "";
	}
	a, aa := getarg(argt, true, true);
	if(t.w != nil) {
		t.w.ref.Inc()
	}
	run(t.w, b, dir, true, aa, a, false);
}


func xexit(*Text, *Text, *Text, bool, bool, string) {
fmt.Println("Exiting?")
	if(row.Clean()){
fmt.Println("Clean")
		close(cexit)
	//	threadexits(nil);
fmt.Println("Sent exit signal")
	}
fmt.Println("Not Exiting?")
}

func del(et * Text, _0 * Text, _1 * Text, flag1  bool, _2  bool, _3 string) {
fmt.Println("Calling del")
	if(et.col==nil || et.w == nil) {
		return;
	}
	if(flag1 || len(et.w.body.file.text)>1 || et.w.Clean(false)) {
		et.col.Close(et.w, true);
	}
}

func cut(et *Text, t *Text, _0 *Text, dosnarf bool, docut bool, _2 []rune) {
	Unimpl()
}

func paste(et *Text, t *Text, _0 *Text, selectall bool, tobody bool, _2 []rune) {
	Unimpl()
}

func get(et *Text, t *Text, argt *Text, flag1 bool, _0 bool, arg []rune) {
	Unimpl()
}
func put(et *Text, _0 *Text, argt *Text, _1 bool, _2 bool, arg []rune) {
	Unimpl()
}

func undo(et *Text, _0 *Text, _1 *Text, flag1 bool, _2 bool, _3 []rune) {
	Unimpl()
}

func run(win *Window, s []rune, rdir string, newns bool, argaddr string, xarg string, iseditcmd bool) {
Untested()
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

	// mustn't block here because must be ready to answer mount() call in run() 

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

func runproc(win *Window, s []rune, rdir string, newns bool, argaddr string, xarg string, c *Command, cpid chan int, iseditcmd bool) {
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

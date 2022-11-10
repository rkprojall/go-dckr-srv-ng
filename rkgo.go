package rkgo
import (
    "fmt"
    "os"
    "os/signal"
    "time"
    "strconv"
    "strings"
    "syscall"
    "net"
)

type remember struct {
     file string
     stamp int64
     address []int
     amount int
}

const MAX_FILES = 4096

func RK_CNT(cond bool, t int, f int, m []int) (res bool) {
        res = cond
	if cond {
	   m[t]++;
        } else {
	  m[f]++;
	}
	return cond;
}

func RK_TRUE(cond bool,m []int,t int, i int) (res bool) {
     res = cond;
     if cond {
     	m[t]+=i;
     }
     return;
}

func RK_MI(m []int, t int) (res bool) {
    res = false;
    m[t] = 0;
    return res
}

var book[MAX_FILES] remember;
var current int = 0
var rkserver = ""

func RK_check_in(file string, stamp int64, amount int) (address []int){
       address = make([]int, amount)
       var entry remember
       entry.file = file
       entry.stamp = stamp
       entry.address = address
       entry.amount = amount
       if current==0 {
	        go func() {
		   sigc := make(chan os.Signal, 1)
		   signal.Notify(sigc)
		   for {
		        s := <-sigc
			//fmt.Println("Got signal:", s)
		        RK_check_out();
			if (s==syscall.SIGINT || s==syscall.SIGTERM) {
			   os.Exit(0)
                        }
	           }
                 }()

		go func() {
		   time.Sleep(5 * time.Second)
		   for {
		       RK_check_out();
       		       time.Sleep(20 * time.Second)
                   }
                }();
       }
       if current < MAX_FILES {
	       book[current] = entry  
	       current++
	   } else {
	   fmt.Println("More than ", MAX_FILES, " instrumented, adjust MAX_FILES in rkgo.go");
       } 
       return address
}


func RK_check_out() {
    content:=""
    for k:=0; k < current; k++ {
    	zero := 1
	entry := book[k];
       for ii := 0 ; ii < entry.amount; ii++ {
	   if entry.address[ii] != 0 {
	       zero = 0
	       ii = entry.amount
	   }
        }
        if zero == 0 {
	       content+=fmt.Sprintf("{\"time\":%d,",time.Now().Unix())
               hostname, e := os.Hostname()
               if e != nil {
                   hostname = "localhost"
               }
               content+=fmt.Sprintf("\"hostname\":\"%s\",",hostname)
               content+=fmt.Sprintf("\"raw\":\"%s\",",entry.file)
               content+="\"stamp\":0,"
               content+="\"cnt\":["
           first := 0
           for i := 0; i < entry.amount; i++ {
               if first != 0 {
	           content+=","
               } else {
                   first = 1
               }
	       content+=fmt.Sprintf("%d",entry.address[i])
               entry.address[i] = 0
           }
	   content+="]}\n"
	}
    }
    if len(content) > 0 {
	if rkserver=="" {
	    rkserver="localhost:9999"
            _, found := os.LookupEnv("RKSERVER")
	    if found {
		rkserver = os.Getenv("RKSERVER")
            } else {
		_, found := os.LookupEnv("HOME")
		if found { 
		    dat, err := os.ReadFile(os.Getenv("HOME")+"/.rkserver")
		    if err == nil {
			lines:= strings.Split(string(dat),"\n")
			rkserver=lines[0];
		    }
		}
	    }
	}
	_push(rkserver,content);
    }
}

func  _push(server string, content string) {
    tcpAddr, err := net.ResolveTCPAddr("tcp", server)
    if err != nil {
        println("ResolveTCPAddr failed:", err.Error())
	return
    }
    conn, err := net.DialTCP("tcp", nil, tcpAddr)
    if err != nil {
        println(err.Error())
	return
    }
    defer conn.Close()
    length := strconv.Itoa(len(content))
    header :="POST / HTTP/1.1\r\n"
                header+="Host: "+server+"\r\n"
                header+="User-Agent: rkgo/2.0.11\r\n"
                header+="Referer: http://localhost/\r\n"
                header+="Accept: */*\r\n"
                header+="Content-Type: text/plain\r\n"
                header+="Content-Length: "+length+"\r\n"
                header+="\r\n"

    _, err = conn.Write([]byte(header+content))
    if err != nil {
        println(err.Error())
	return
    }
    reply := make([]byte, 1024)
    _, err = conn.Read(reply)
    if err != nil {
      //  println("Read from server failed:", err.Error())
    }

    //println("reply from server=", string(reply))



}

// atexit → use "C" cgo, callback then to __rk_check_out()
// signal handlers, similar as in C 

package handlers

import (
	crypto_rand "crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"golang.org/x/crypto/bcrypt"
)

// PlayerSvc is the package-level player service for handler use.
var PlayerSvc *services.PlayerService

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

var disposableEmailDomains = map[string]bool{
	"0-mail.com": true, "10minutemail.com": true, "10minutemail.net": true, "20minutemail.com": true,
	"33mail.com": true, "3d-painting.com": true, "4warding.com": true, "4warding.net": true, "4warding.org": true,
	"60minutemail.com": true, "675hosting.com": true, "675hosting.net": true, "675hosting.org": true,
	"6paq.com": true, "6url.com": true, "75hosting.com": true, "75hosting.net": true, "75hosting.org": true,
	"7tags.com": true, "9ox.net": true, "a-bc.net": true, "a45.in": true, "abyssmail.com": true,
	"adbetron.com": true, "afrobacon.com": true, "ajaxapp.net": true, "amilegit.com": true,
	"amiri.net": true, "amiriindustries.com": true, "anappthat.com": true, "anonbox.net": true,
	"anonmails.de": true, "anonymail.dk": true, "anonymbox.com": true, "antichef.com": true,
	"antichef.net": true, "antireg.ru": true, "antispam.de": true, "antispammail.de": true,
	"armyspy.com": true, "artman-conception.com": true, "avoidc.com": true, "baxomale.ht.cx": true,
	"beddly.com": true, "bigprofessor.so": true, "binkmail.com": true, "bio-muesli.net": true,
	"blogmyway.org": true, "bobmail.info": true, "bootybay.de": true, "boun.cr": true,
	"bouncr.com": true, "boxformail.in": true, "breakthru.com": true, "brefmail.com": true,
	"brennendesreich.de": true, "broadbandninja.com": true, "bsnow.net": true, "bu.mintemail.com": true,
	"buffemail.com": true, "bugmenever.com": true, "bumpymail.com": true, "bund.us": true,
	"bundes-li.ga": true, "burnthespam.info": true, "burstmail.info": true, "buxap.com": true,
	"buyusedlibrarybooks.org": true, "byom.de": true, "c2.hu": true, "cachedot.net": true,
	"casualdx.com": true, "cellurl.com": true, "centermail.com": true, "centermail.net": true,
	"chammy.info": true, "cheaphub.net": true, "cheatmail.de": true, "chogmail.com": true,
	"choicemail1.com": true, "clixser.com": true, "clrmail.com": true, "cmail.com": true,
	"cmail.net": true, "cmail.org": true, "coldemail.info": true, "consumerriot.com": true,
	"cool.fr.nf": true, "correo.blogos.net": true, "cosmorph.com": true, "courrieltemporaire.com": true,
	"crapmail.org": true, "crazymailing.com": true, "cubiclink.com": true, "curryworld.de": true,
	"cust.in": true, "cuvox.de": true, "d3p.dk": true, "dacoolest.com": true, "daintly.com": true,
	"dandikmail.com": true, "dayrep.com": true, "dbunker.com": true, "dcemail.com": true,
	"deadaddress.com": true, "deadspam.com": true, "debuggingguy.com": true, "delikkt.de": true,
	"despam.it": true, "devnullmail.com": true, "digdig.org": true, "digitalsanctuary.com": true,
	"discard.email": true, "discardmail.com": true, "discardmail.de": true, "disposable-email.ml": true,
	"disposableaddress.com": true, "disposableemailaddresses.com": true, "disposableinbox.com": true,
	"dispose.it": true, "disposeamail.com": true, "disposemail.com": true, "dispostable.com": true,
	"dmarc.rocks": true, "dodgeit.com": true, "dodsi.com": true, "doiea.com": true,
	"domozmail.com": true, "donemail.ru": true, "dontsendmespam.de": true, "dotmsg.com": true,
	"dropmail.me": true, "dzfh.de": true, "e-mail.com": true, "e-mail.org": true,
	"e4ward.com": true, "easytrashmail.com": true, "einrot.com": true, "einrot.de": true,
	"eintagsmail.de": true, "email-temp.com": true, "email60.com": true, "emailage.com": true,
	"emaildienst.de": true, "emailfake.com": true, "emailgenerator.de": true, "emailgo.de": true,
	"emailias.com": true, "emailigo.de": true, "emailinfive.com": true, "emaillime.com": true,
	"emailmiser.com": true, "emailondeck.com": true, "emailproxsy.com": true, "emailtemporanea.com": true,
	"emailtemporanea.net": true, "emailtemporar.ro": true, "emailtemporario.com.br": true, "emailthe.net": true,
	"emailtmp.com": true, "emailto.de": true, "emailwarden.com": true, "emailxfer.com": true,
	"emeil.in": true, "emeil.ir": true, "emil.com": true, "emkei.cf": true, "emkei.ga": true,
	"emkei.gq": true, "emkei.ml": true, "emkei.tk": true, "emlpro.com": true, "emltrk.com": true,
	"empireanime.xyz": true, "enterto.com": true, "ephemail.net": true, "erasedebt.gq": true,
	"etranquil.com": true, "etranquil.net": true, "etranquil.org": true, "evopo.com": true,
	"explodemail.com": true, "eyepaste.com": true, "ez.lv": true, "fake-box.com": true,
	"fakeinbox.com": true, "fakeinformation.com": true, "fakemail.fr": true, "fakemailgenerator.com": true,
	"fakemailz.com": true, "fammix.com": true, "fansworldwide.de": true, "fantasymail.de": true,
	"fastacura.com": true, "fastchevy.com": true, "fastchrysler.com": true, "fastkawasaki.com": true,
	"fastmazda.com": true, "fastmitsubishi.com": true, "fastnissan.com": true, "fastsubaru.com": true,
	"fastsuzuki.com": true, "fasttoyota.com": true, "fastyamaha.com": true, "fightallspam.com": true,
	"filzmail.com": true, "fivemail.de": true, "fleckens.hu": true, "flemail.ru": true,
	"flyspam.com": true, "footard.com": true, "forgetmail.com": true, "frapmail.com": true,
	"free-temp.net": true, "freebulgaria.bg": true, "freemails.cf": true, "freemails.ga": true,
	"freemails.ml": true, "freundin.ru": true, "friendlymail.co.uk": true, "front14.org": true,
	"fsagc.xyz": true, "fuckingduh.com": true, "fudgerub.com": true, "fuwamofu.com": true,
	"fw2.me": true, "fw6m0bd.com": true, "gafy.net": true, "gamno.config.work": true,
	"garliclife.com": true, "gelitik.in": true, "get-mail.cf": true, "get-mail.ga": true,
	"get-mail.ml": true, "get-mail.tk": true, "getairmail.com": true, "getmails.eu": true,
	"getnada.com": true, "getonemail.com": true, "getsimpleemail.com": true, "giantmail.de": true,
	"gishpuppy.com": true, "gmal.com": true, "gmial.com": true, "go2usa.info": true,
	"goat.si": true, "gorillaswithdirtyarmpits.com": true, "gotmail.com": true, "gotmail.net": true,
	"gotmail.org": true, "gowikibooks.com": true, "gowikicampus.com": true, "gowikicars.com": true,
	"gowikifilms.com": true, "gowikigames.com": true, "gowikimusic.com": true, "gowikimusic.great-host.in": true,
	"gowikinetwork.com": true, "gowikitravel.com": true, "gowikitv.com": true, "grandmamail.com": true,
	"great-host.in": true, "greencafe24.com": true, "greendike.com": true, "greenhousemail.com": true,
	"greensloth.com": true, "grr.la": true, "gsrv.co.uk": true, "guerrillamail.biz": true,
	"guerrillamail.com": true, "guerrillamail.de": true, "guerrillamail.net": true, "guerrillamail.org": true,
	"guerrillamailblock.com": true, "gustr.com": true, "h.mintemail.com": true, "h8s.org": true,
	"hacccc.com": true, "haltospam.com": true, "harakirimail.com": true, "hartbot.de": true,
	"hat-geld.de": true, "hatespam.org": true, "herp.in": true, "hidemail.de": true,
	"hidzz.com": true, "hmamail.com": true, "hochsitze.com": true, "hopemail.biz": true,
	"hotpop.com": true, "hulapla.de": true, "hushmail.com": true, "ieatspam.eu": true,
	"ieatspam.info": true, "ihateyoualot.info": true, "iheartspam.org": true, "ikbenspamvrij.nl": true,
	"imails.info": true, "imgof.com": true, "imgv.de": true, "immo-gerance.info": true,
	"imstations.com": true, "inbax.tk": true, "inbox.si": true, "inboxalias.com": true,
	"inboxbear.com": true, "inboxclean.com": true, "inboxclean.org": true, "inboxdesign.me": true,
	"inboxed.im": true, "inboxed.pw": true, "inboxkitten.com": true, "inboxproxy.com": true,
	"incognitomail.com": true, "incognitomail.net": true, "incognitomail.org": true, "indieclad.com": true,
	"infocom.zp.ua": true, "insorg-mail.info": true, "instant-mail.de": true, "ip6.li": true,
	"irish2me.com": true, "iwi.net": true, "j6j2y0.com": true, "jamit.com.au": true,
	"jetable.com": true, "jetable.fr.nf": true, "jetable.net": true, "jetable.org": true,
	"jnxjn.com": true, "jobbikszimpatizans.hu": true, "jourrapide.com": true, "jsrsolutions.com": true,
	"junk1e.com": true, "jwk4227uf.com": true, "kasmail.com": true, "kaspop.com": true,
	"keepmymail.com": true, "kemptvillebaseball.com": true, "kiani.com": true, "killmail.com": true,
	"killmail.net": true, "kimsdisk.com": true, "kingsq.ga": true, "kiois.com": true,
	"kiryubox.cu.cc": true, "klassmaster.com": true, "klassmaster.net": true, "klzlk.com": true,
	"kook.ml": true, "koszmail.pl": true, "kuku.cf": true, "kulturbetrieb.info": true,
	"kurzepost.de": true, "l33r.eu": true, "lackmail.net": true, "lakqs.com": true,
	"lawlita.com": true, "leeching.net": true, "letthemeatspam.com": true, "lhsdv.com": true,
	"lifebyfood.com": true, "link2mail.net": true, "linkedintuts2016.pw": true, "litedrop.com": true,
	"loadby.us": true, "login-email.ml": true, "loh.pp.ua": true, "loin.in": true,
	"lol.ovpn.to": true, "lookugly.com": true, "lopl.co.cc": true, "lortemail.dk": true,
	"lovemeleaveme.com": true, "lroid.com": true, "lron.us": true, "luckymail.org": true,
	"maboard.com": true, "macr2.com": true, "mail-card.net": true, "mail-easy.fr": true,
	"mail-filter.com": true, "mail-temporaire.fr": true, "mail-tester.com": true, "mail.by": true,
	"mail.illistno.com": true, "mail.zp.ua": true, "mail0.ga": true, "mail114.net": true,
	"mail1a.de": true, "mail21.cc": true, "mail2rss.org": true, "mail333.com": true,
	"mail4trash.com": true, "mail666.ru": true, "mail707.com": true, "mail72.com": true,
	"mailaboki.com": true, "mailalias.ga": true, "mailback.com": true, "mailbidon.com": true,
	"mailbiz.biz": true, "mailblocks.com": true, "mailbox72.biz": true, "mailbox80.biz": true,
	"mailbucket.org": true, "mailcat.biz": true, "mailcatch.com": true, "mailde.de": true,
	"mailde.info": true, "maildrop.cc": true, "maildrop.cf": true, "maildrop.ga": true,
	"maildrop.gq": true, "maildrop.ml": true, "maildu.de": true, "maildx.com": true,
	"maileater.com": true, "mailexpire.com": true, "mailf5.com": true, "mailfa.tk": true,
	"mailfall.com": true, "mailforspam.com": true, "mailfree.ga": true, "mailfree.ml": true,
	"mailfreeonline.com": true, "mailfs.com": true, "mailgogo.tk": true, "mailguard.me": true,
	"mailgutter.com": true, "mailhazard.com": true, "mailhazard.us": true, "mailhex.com": true,
	"mailhz.me": true, "mailimate.com": true, "mailin8r.com": true, "mailinater.com": true,
	"mailinator.com": true, "mailinator.net": true, "mailinator.org": true, "mailinator.us": true,
	"mailinator2.com": true, "mailincubator.com": true, "mailismagic.com": true, "mailita.tk": true,
	"mailjunk.cf": true, "mailjunk.ga": true, "mailjunk.gq": true, "mailjunk.ml": true,
	"mailjunk.tk": true, "mailmate.com": true, "mailme.ir": true, "mailme.lv": true,
	"mailmetrash.com": true, "mailmoat.com": true, "mailmoth.com": true, "mailms.com": true,
	"mailnator.com": true, "mailnesia.com": true, "mailnull.com": true, "mailorg.org": true,
	"mailpick.biz": true, "mailpooch.com": true, "mailproxsy.com": true, "mailquack.com": true,
	"mailrock.biz": true, "mailscrap.com": true, "mailseal.de": true, "mailshell.com": true,
	"mailsiphon.com": true, "mailslapping.com": true, "mailslite.com": true, "mailspam.xyz": true,
	"mailspam.tk": true, "mailt.net": true, "mailtemp.fr": true, "mailtemporaire.com": true,
	"mailtemporaire.fr": true, "mailtmp.com": true, "mailto.space": true, "mailtome.de": true,
	"mailtothis.com": true, "mailtraps.com": true, "mailtrash.net": true, "mailtrix.net": true,
	"mailtv.net": true, "mailtv.tv": true, "mailwithyou.com": true, "mailzi.ru": true,
	"mailzilla.com": true, "mailzilla.org": true, "manifestgenerator.com": true, "manybrain.com": true,
	"mbx.cc": true, "mciek.com": true, "meantinc.com": true, "mega.zik.dj": true,
	"meinspamschutz.de": true, "meltmail.com": true, "messagebeamer.de": true, "mezimages.net": true,
	"mfsa.ru": true, "mierdamail.com": true, "migmail.net": true, "migmail.pl": true,
	"migumail.com": true, "mindmail.ga": true, "ministry-of-silly-walks.de": true, "mintemail.com": true,
	"misterpinball.de": true, "mjukglass.nu": true, "moakt.com": true, "moburl.com": true,
	"mohmal.com": true, "momentics.ru": true, "moncourrier.fr.nf": true, "monemail.fr.nf": true,
	"monmail.fr.nf": true, "monumentmail.com": true, "morahdsl.cf": true, "mt2009.com": true,
	"mt2014.com": true, "mt2015.com": true, "muellemail.com": true, "muellmail.com": true,
	"mundomail.net": true, "mutant.me": true, "mvrht.com": true, "mvrht.net": true,
	"mwarrior.org": true, "mx0.wwwnew.eu": true, "my10minutemail.com": true, "mycard.net.ua": true,
	"mycleaninbox.net": true, "mydemo.equipment": true, "myecho.es": true, "mymail-in.net": true,
	"mymail90.com": true, "mymailoasis.com": true, "mynetstore.de": true, "mypartyclip.de": true,
	"myphantomemail.com": true, "mysamp.de": true, "myspaceinc.com": true, "myspaceinc.net": true,
	"myspaceinc.org": true, "myspacepimpedup.com": true, "myspamless.com": true, "mytemp.email": true,
	"mytempemail.com": true, "mytrashmail.com": true, "n4paml3w6qn.com": true, "nabuma.com": true,
	"negated.com": true, "neomailbox.com": true, "nepwk.com": true, "nervmich.net": true,
	"nervtmich.net": true, "netmails.com": true, "netmails.net": true, "netzidiot.de": true,
	"neverbox.com": true, "nice-4u.com": true, "nincsmail.com": true, "nincsmail.hu": true,
	"nnh.com": true, "no-spam.ws": true, "noblepioneer.com": true, "nobulk.com": true,
	"noclickemail.com": true, "nogmailspam.info": true, "nomail.pw": true, "nomail.xl.cx": true,
	"nomail2me.com": true, "nomorespamemails.com": true, "nonspam.eu": true, "nonspammer.de": true,
	"noref.in": true, "nospam.ze.tc": true, "nospam4.us": true, "nospamfor.us": true,
	"nospammail.net": true, "nospamthanks.info": true, "notmailinator.com": true, "nowmymail.com": true,
	"ntlhelp.net": true, "nurfuerspam.de": true, "nus.edu.sg": true, "o2stk.org": true,
	"objectmail.com": true, "obobbo.com": true, "odaymail.com": true, "odnorazovoe.ru": true,
	"one-time.email": true, "oneoffemail.com": true, "oneoffmail.com": true, "onewaymail.com": true,
	"online.ms": true, "oopi.org": true, "opayq.com": true, "ordinaryamerican.net": true,
	"otherinbox.com": true, "ourklips.com": true, "outlawspam.com": true, "ovpn.to": true,
	"owlpic.com": true, "pancakemail.com": true, "paplease.com": true, "pcusers.otherinbox.com": true,
	"pepbot.com": true, "pfui.ru": true, "pimpedupmyspace.com": true, "pjjkp.com": true,
	"plexolan.de": true, "poczta.onet.pl": true, "politikerclub.de": true, "pooae.com": true,
	"pookmail.com": true, "postacin.com": true, "powerencry.com": true, "privacy.net": true,
	"privy-mail.com": true, "privymail.de": true, "proxymail.eu": true, "prtnx.com": true,
	"prtz.eu": true, "punkass.com": true, "put2.net": true, "putthisinyourspamdatabase.com": true,
	"pwrby.com": true, "qbi.kr": true, "quickemail.info": true, "quickinbox.com": true,
	"quickmail.nl": true, "rcpt.at": true, "reallymymail.com": true, "realtyalerts.ca": true,
	"receiveee.com": true, "recode.me": true, "recursor.net": true, "regbypass.com": true,
	"rejectmail.com": true, "remail.cf": true, "remail.ga": true, "rmqkr.net": true,
	"royal.net": true, "rppkn.com": true, "rtrtr.com": true, "s0ny.net": true,
	"safe-mail.net": true, "safersignup.de": true, "safetymail.info": true, "safetypost.de": true,
	"sandelf.de": true, "saynotospams.com": true, "schafmail.de": true, "schrott-email.de": true,
	"secretemail.de": true, "secure-mail.biz": true, "selfdestructingmail.com": true, "sendspamhere.com": true,
	"sharklasers.com": true, "shieldedmail.com": true, "shiftmail.com": true, "shitmail.de": true,
	"shitmail.org": true, "shitware.nl": true, "shmeriously.com": true, "shortmail.net": true,
	"shotmail.ru": true, "showslow.de": true, "sibmail.com": true, "sinnlos-mail.de": true,
	"siteposter.net": true, "skeefmail.com": true, "slapsfromlastnight.com": true,
	"slaskpost.se": true, "slipry.net": true, "slopsbox.com": true, "slusare.info": true,
	"smashmail.de": true, "smellfear.com": true, "smellrear.com": true, "snakemail.com": true,
	"sneakemail.com": true, "sneakmail.de": true, "snkmail.com": true, "sofimail.com": true,
	"soisz.com": true, "solvemail.info": true, "soodonims.com": true, "spam.la": true,
	"spam.su": true, "spam4.me": true, "spamail.de": true, "spamarrest.com": true,
	"spamavert.com": true, "spambob.com": true, "spambob.net": true, "spambob.org": true,
	"spambog.com": true, "spambog.de": true, "spambog.net": true, "spambog.ru": true,
	"spambox.info": true, "spambox.irishspringrealty.com": true, "spambox.us": true,
	"spamcannon.com": true, "spamcannon.net": true, "spamcero.com": true, "spamcon.org": true,
	"spamcorptastic.com": true, "spamcowboy.com": true, "spamcowboy.net": true, "spamcowboy.org": true,
	"spamday.com": true, "spamdecoy.net": true, "spamex.com": true, "spamfighter.cf": true,
	"spamfighter.ga": true, "spamfighter.gq": true, "spamfighter.ml": true, "spamfighter.tk": true,
	"spamfree.eu": true, "spamfree24.com": true, "spamfree24.de": true, "spamfree24.eu": true,
	"spamfree24.info": true, "spamfree24.net": true, "spamfree24.org": true, "spamgoes.in": true,
	"spamgourmet.com": true, "spamgourmet.net": true, "spamgourmet.org": true, "spamherelots.com": true,
	"spamhereplease.com": true, "spamhole.com": true, "spamify.com": true, "spaminator.de": true,
	"spamkill.info": true, "spaml.com": true, "spaml.de": true, "spammotel.com": true,
	"spamobox.com": true, "spamoff.de": true, "spamproof.be": true, "spamsalad.in": true,
	"spamsphere.com": true, "spamspot.com": true, "spamthis.co.uk": true, "spamthisplease.com": true,
	"spamtrail.com": true, "spamtrap.ro": true, "speedgaus.net": true, "sperma.cf": true,
	"spikio.com": true, "spoofmail.de": true, "squizzy.de": true, "ssoia.com": true,
	"startfu.com": true, "stealthypost.org": true, "stexsy.com": true, "stinkefinger.net": true,
	"stop-my-spam.cf": true, "stop-my-spam.ga": true, "stop-my-spam.ml": true, "stop-my-spam.tk": true,
	"streetwisemail.com": true, "stuffmail.de": true, "super-auswahl.de": true, "supergreatmail.com": true,
	"supermailer.jp": true, "superrito.com": true, "suremail.info": true, "sute.jp": true,
	"svk.jp": true, "tafmail.com": true, "taglead.com": true, "talkinator.com": true,
	"tapchicuoihoi.com": true, "techemail.com": true, "teerest.com": true, "temp-mail.net": true,
	"tempail.com": true, "tempalias.com": true, "tempe-mail.com": true, "tempemail.biz": true,
	"tempemail.co.za": true, "tempemail.com": true, "tempemail.net": true, "tempinbox.co.uk": true,
	"tempinbox.com": true, "tempmail.co": true, "tempmail.de": true, "tempmail.eu": true,
	"tempmail.it": true, "tempmail.net": true, "tempmail.ninja": true, "tempmail.org": true,
	"tempmail.pro": true, "tempmail.plus": true, "tempmail.us": true, "tempmail2.com": true,
	"tempmaildemo.com": true, "tempmailer.com": true, "tempmailer.de": true, "tempmailid.com": true,
	"tempomail.com": true, "temporarily.de": true, "temporarioemail.com.br": true, "temporaryemail.net": true,
	"temporaryemail.us": true, "temporaryforwarding.com": true, "temporaryinbox.com": true, "temporarymailaddress.com": true,
	"tempsky.com": true, "tempthe.net": true, "tempymail.com": true, "thanksnospam.info": true,
	"thankyou2010.com": true, "thisisnotmyrealemail.com": true, "thismail.net": true,
	"throwawayemailaddress.com": true, "throwawaymail.com": true, "throya.com": true,
	"tilien.com": true, "tittbit.in": true, "tizi.com": true, "tmail.com": true,
	"tmailinator.com": true, "toiea.com": true, "toomail.biz": true, "topranklist.de": true,
	"tradermail.info": true, "trash-amil.com": true, "trash-mail.at": true, "trash-mail.cf": true,
	"trash-mail.com": true, "trash-mail.de": true, "trash-mail.ga": true, "trash-mail.gq": true,
	"trash-mail.ml": true, "trash-mail.tk": true, "trash2009.com": true, "trash2010.com": true,
	"trash2011.com": true, "trashcanmail.com": true, "trashdevil.com": true, "trashdevil.de": true,
	"trashemail.de": true, "trashimail.de": true, "trashinbox.com": true, "trashmail.at": true,
	"trashmail.com": true, "trashmail.de": true, "trashmail.io": true, "trashmail.me": true,
	"trashmail.net": true, "trashmail.org": true, "trashmail.ws": true, "trashmailer.com": true,
	"trashymail.com": true, "trashymail.net": true, "trbvm.com": true, "trbvn.com": true,
	"trbvo.com": true, "trialmail.de": true, "trimsj.com": true, "trollproject.com": true,
	"tryalert.com": true, "turoid.com": true, "twinmail.de": true, "tyldd.com": true,
	"uggsrock.com": true, "umail.net": true, "uroid.com": true, "us.af": true,
	"venompen.com": true, "veryrealemail.com": true, "vidchart.com": true, "viditag.com": true,
	"viewcastmedia.com": true, "viewcastmedia.net": true, "viewcastmedia.org": true,
	"visal168.com": true, "visal168.cf": true, "visal168.ga": true, "visal168.gq": true,
	"visal168.ml": true, "visal168.tk": true, "vomoto.com": true, "vpn.st": true,
	"vsimcard.com": true, "vubby.com": true, "wasteland.rfc822.org": true, "webcontact-france.eu": true,
	"webm4il.info": true, "wegas.ru": true, "wegwerf-email-addressen.de": true, "wegwerf-email-adressen.de": true,
	"wegwerf-email.de": true, "wegwerf-email.net": true, "wegwerf-emails.de": true, "wegwerfadresse.de": true,
	"wegwerfemail.com": true, "wegwerfemail.de": true, "wegwerfemail.info": true, "wegwerfemail.net": true,
	"wegwerfemail.org": true, "wegwerfmail.de": true, "wegwerfmail.info": true, "wegwerfmail.net": true,
	"wegwerfmail.org": true, "wetrainbayarea.com": true, "wetrainbayarea.org": true,
	"wh4f.org": true, "whyspam.me": true, "willhackforfood.biz": true, "willselfdestruct.com": true,
	"winemaven.info": true, "wizaz.com": true, "wronghead.com": true, "wuzup.net": true,
	"xagloo.com": true, "xemaps.com": true, "xents.com": true, "xmaily.com": true,
	"xoxy.net": true, "yep.it": true, "yogamaven.com": true, "yopmail.com": true,
	"yopmail.fr": true, "yopmail.net": true, "yopmail.org": true, "you-spam.com": true,
	"ypmail.webarnak.fr.eu.org": true, "yuurok.com": true, "zebins.com": true,
	"zebins.eu": true, "zehnminuten.de": true, "zehnminutenmail.de": true, "zippymail.info": true,
	"zoaxe.com": true, "zoemail.com": true, "zoemail.net": true, "zoemail.org": true,
	"zomg.info": true, "zxcv.com": true, "zxcvbnm.com": true, "ezehe.com": true,
	"tempmail.cn": true, "gmail.com.de": true, "emailfake.org": true,
	"tmpmail.org": true, "tmp.email": true, "10m.email": true, "linshiyou.com": true,
	"moakt.cc": true, "emailnator.com": true, "eyso.net": true, "mytrashmailer.com": true,
	"mailgoy.com": true, "mailsac.com": true, "10minutemail.info": true, "anotherdomain.com": true,
	"mybink.io": true, "nada.email": true, "nada.ltd": true, "gmailx.de": true,
}

func isDisposableEmail(email string) bool {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	domain := strings.ToLower(parts[1])
	return disposableEmailDomains[domain]
}

func isValidUsername(username string) bool {
	for _, r := range username {
		if !unicode.IsLetter(r) && r != '-' && !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// CreatePlayerRequest represents the incoming registration payload
type CreatePlayerRequest struct {
	Username       string `json:"username" binding:"required,min=3,max=30"`
	Password       string `json:"password" binding:"required,min=6,max=50"`
	StartingRegion string `json:"starting_region"`
	SetVitalNow    bool   `json:"set_vital_now"`
	Email          string `json:"email" binding:"required,email"`
	ReferralCode   string `json:"referral_code"`
	AttackGuID     string `json:"attack_gu_id"`
	DefenseGuID    string `json:"defense_gu_id"`
	VitalGuChoice  string `json:"vital_gu_choice"`
	OTP            string `json:"otp" binding:"required"`
}

// CreatePlayerResponse represents the output payload for registration/login
type CreatePlayerResponse struct {
	Player models.Player `json:"player"`
	Token  string        `json:"token,omitempty"`
}

const authCookieName = "gu_auth_token"
const authCookieMaxAge = 7 * 24 * 60 * 60

func authCookieDomain() string {
	return os.Getenv("AUTH_COOKIE_DOMAIN")
}

func authCookieSecure(c *gin.Context, isSameSiteNone bool) bool {
	secureEnv := os.Getenv("AUTH_COOKIE_SECURE")
	if secureEnv != "" {
		return strings.EqualFold(secureEnv, "true")
	}

	if isSameSiteNone {
		return true // SameSite=None strictly requires Secure=true in modern browsers
	}

	return c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https"
}

func setAuthCookie(c *gin.Context, token string) {
	sameSite := http.SameSiteLaxMode

	// Tự động phát hiện yêu cầu cross-origin/cross-site
	origin := c.Request.Header.Get("Origin")
	corsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")

	originHost := ""
	if origin != "" {
		if u, err := url.Parse(origin); err == nil {
			originHost = u.Hostname()
		}
	} else if corsEnv != "" {
		firstCors := corsEnv
		if strings.Contains(corsEnv, ",") {
			firstCors = strings.Split(corsEnv, ",")[0]
		}
		if u, err := url.Parse(firstCors); err == nil {
			originHost = u.Hostname()
		}
	}

	requestHost := c.Request.Host
	if strings.Contains(requestHost, ":") {
		requestHost = strings.Split(requestHost, ":")[0]
	}

	// Nếu host của frontend khác host của backend, chúng là cross-site
	// Ngoại lệ: Nếu cả hai đều chạy trên localhost, SameSite Lax vẫn được chấp nhận
	if originHost != "" && originHost != requestHost {
		if originHost == "localhost" && requestHost == "localhost" {
			sameSite = http.SameSiteLaxMode
		} else {
			sameSite = http.SameSiteNoneMode
		}
	}

	switch strings.ToLower(os.Getenv("AUTH_COOKIE_SAMESITE")) {
	case "strict":
		sameSite = http.SameSiteStrictMode
	case "none":
		sameSite = http.SameSiteNoneMode
	case "lax":
		sameSite = http.SameSiteLaxMode
	}

	c.SetSameSite(sameSite)
	c.SetCookie(authCookieName, token, authCookieMaxAge, "/", authCookieDomain(), authCookieSecure(c, sameSite == http.SameSiteNoneMode), true)

	// Set CSRF token cookie (non-HttpOnly so JS can read it for header)
	csrfToken, err := middleware.GenerateCSRFToken()
	if err == nil {
		middleware.SetCSRFCookie(c, csrfToken)
	}
}

func clearAuthCookie(c *gin.Context) {
	sameSite := http.SameSiteLaxMode
	origin := c.Request.Header.Get("Origin")
	corsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")
	if (origin != "" && !strings.Contains(origin, "localhost") && !strings.Contains(origin, c.Request.Host)) ||
		(corsEnv != "" && !strings.Contains(corsEnv, "localhost") && corsEnv != "*") {
		sameSite = http.SameSiteNoneMode
	}

	c.SetSameSite(sameSite)
	c.SetCookie(authCookieName, "", -1, "/", authCookieDomain(), authCookieSecure(c, sameSite == http.SameSiteNoneMode), true)
	middleware.ClearCSRFCookie(c)
}

// RegisterPlayer handles registering a new player, hashing password, and rolling their aptitude
func RegisterPlayer(c *gin.Context) {
	var req CreatePlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) < 3 || len(req.Username) > 30 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hiệu phải từ 3 đến 30 ký tự!"})
		return
	}
	if !isValidUsername(req.Username) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hiệu chỉ được chứa chữ cái, dấu gạch ngang và khoảng trắng!"})
		return
	}

	validRegions := map[string]bool{
		"Southern_Border":   true,
		"Central_Continent": true,
		"Northern_Plains":   true,
		"Western_Desert":    true,
		"Eastern_Sea":       true,
	}
	if req.StartingRegion == "" {
		req.StartingRegion = "Southern_Border"
	}
	if !validRegions[req.StartingRegion] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Starting region không hợp lệ"})
		return
	}

	// Email format verification
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Định dạng Email không hợp lệ!"})
		return
	}
	if isDisposableEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email tạm thời không được chấp nhận. Vui lòng dùng email thật!"})
		return
	}

	// Validate selected starting Gu
	if req.AttackGuID == "" {
		req.AttackGuID = "GU_MOONLIGHT"
	}
	if req.DefenseGuID == "" {
		req.DefenseGuID = "GU_STONE_SKIN"
	}

	validAttackGus := map[string]bool{
		"GU_MOONLIGHT":  true,
		"GU_STREAM":     true,
		"GU_BONE_SPEAR": true,
	}
	validDefenseGus := map[string]bool{
		"GU_GREEN_SILK": true,
		"GU_STONE_SKIN": true,
		"GU_IRON_SKIN":  true,
		"GU_BEAST_SKIN": true,
		"GU_FISH_SCALE": true,
	}

	if !validAttackGus[req.AttackGuID] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng tấn công không hợp lệ"})
		return
	}
	if !validDefenseGus[req.DefenseGuID] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng phòng thủ không hợp lệ"})
		return
	}

	// 1. Hash password with bcrypt
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to secure password"})
		return
	}
	passwordHash := string(hashedBytes)

	// 2. Gacha rolling for aptitude
	roll := rand.Intn(100)
	var aptitudeType string
	var aptitudePercent int

	if roll < 10 {
		aptitudeType = "Giáp"
		aptitudePercent = 80 + rand.Intn(20) // 80 - 99
	} else if roll < 30 {
		aptitudeType = "Ất"
		aptitudePercent = 60 + rand.Intn(20) // 60 - 79
	} else if roll < 60 {
		aptitudeType = "Bính"
		aptitudePercent = 40 + rand.Intn(20) // 40 - 59
	} else {
		aptitudeType = "Đinh"
		aptitudePercent = 20 + rand.Intn(20) // 20 - 39
	}

	// Map aptitude type to numeric grade (1 = A/Giáp .. 4 = D/Đinh)
	aptitudeGrade := 4
	switch aptitudeType {
	case "Giáp":
		aptitudeGrade = 1
	case "Ất":
		aptitudeGrade = 2
	case "Bính":
		aptitudeGrade = 3
	case "Đinh":
		aptitudeGrade = 4
	}

	maxMana := float64(aptitudePercent)
	currentMana := maxMana

	isAttackVital := false
	isDefenseVital := false

	if req.VitalGuChoice == "ATTACK" {
		isAttackVital = true
	} else if req.VitalGuChoice == "DEFENSE" {
		isDefenseVital = true
	} else if req.VitalGuChoice == "NONE" {
		// both false
	} else {
		// fallback to SetVitalNow
		if req.SetVitalNow {
			isAttackVital = true // default to attack
		}
	}

	attackGuInstanceID := uuid.New().String()
	defenseGuInstanceID := uuid.New().String()

	// 3. Initialize starter inventory for Phase 2
	starterGu := models.GuInventory{
		{
			InstanceID: attackGuInstanceID,
			GuID:       req.AttackGuID,
			Rank:       1,
			Hunger:     100,
			Status:     "ACTIVE",
			Equipped:   true,
			IsVital:    isAttackVital,
		},
		{
			InstanceID: defenseGuInstanceID,
			GuID:       req.DefenseGuID,
			Rank:       1,
			Hunger:     100,
			Status:     "ACTIVE",
			Equipped:   true,
			IsVital:    isDefenseVital,
		},
	}
	equippedGus := models.EquippedGusList{attackGuInstanceID, defenseGuInstanceID}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Verify email OTP inside tx
	var dbOTP string
	var expiresAt time.Time
	err = tx.QueryRow("SELECT otp, expires_at FROM email_otps WHERE email = $1", req.Email).Scan(&dbOTP, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mã xác thực OTP chưa được gửi hoặc đã bị xóa!"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống khi xác thực OTP"})
		}
		return
	}

	if time.Now().After(expiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã OTP đã hết hạn! Vui lòng yêu cầu mã mới."})
		return
	}

	if dbOTP != req.OTP {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã OTP không chính xác!"})
		return
	}

	// Delete OTP to prevent reuse
	_, _ = tx.Exec("DELETE FROM email_otps WHERE email = $1", req.Email)

	// Verify email uniqueness inside tx
	var emailExists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM players WHERE email = $1)", req.Email).Scan(&emailExists)
	if err == nil && emailExists {
		c.JSON(http.StatusConflict, gin.H{"error": "Email này đã được đăng ký trong Cổ Giới!"})
		return
	}

	// Verify username uniqueness inside tx
	var usernameExists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM players WHERE username = $1)", req.Username).Scan(&usernameExists)
	if err == nil && usernameExists {
		c.JSON(http.StatusConflict, gin.H{"error": "Đạo hiệu này đã tồn tại trong Cổ Giới!"})
		return
	}

	// Generate unique referral code
	refCode := "REF-" + strings.ToUpper(uuid.New().String()[:8])

	var player models.Player
	query := `
		INSERT INTO players (username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, lifespan, time_speed_multiplier, email, temp_loot, explore_count, last_explore_reset, soul_hp, max_soul, soul_tier, needs_soul_refine, bio, avatar_url, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, ref_code, needs_onboarding)
		VALUES ($1, $2, 1, 1, $3, $4, $5, $6, $7, 20, 0, 0.0, 0.0, NULL, 0, 100.0, 100.0, '[]'::jsonb, 50, 50, '["50,50"]'::jsonb, $11, '{}'::jsonb, $8, '{"strength": 0}'::jsonb, '{}'::jsonb, NULL, 100.0, 1.0, $9, '{"gu_inventory": [], "materials_inventory": {}, "primeval_stones": 0}'::jsonb, 0, NOW(), 10, 10, 'NHAN_HON', false, 'Hãy viết gì đó về bản thân...', '', 1, 1, '', false, $10, true)
		RETURNING id, username, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, ` + db.GuInventoryAggregate("players.id") + `, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, last_mana_update, created_at, lifespan, time_speed_multiplier, party_id, arena_rank, email, google_id, temp_loot, explore_count, last_explore_reset, aperture_stress, last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine, bio, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, is_admin, ref_code, avatar_url, needs_onboarding
	`

	var activeCombatBytes []byte
	err = tx.QueryRow(query, req.Username, passwordHash, aptitudeType, aptitudePercent, aptitudeGrade, currentMana, maxMana, req.StartingRegion, req.Email, refCode, equippedGus).Scan(
		&player.ID, &player.Username, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones, &player.GuInventory,
		&player.WallProgress, &player.HumanQi, &player.NextTribulationTime, &player.TribulationCount,
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
		&player.EquippedGus, &player.ClearedCells,
		&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries, &activeCombatBytes,
		&player.LastManaUpdate, &player.CreatedAt,
		&player.Lifespan, &player.TimeSpeedMultiplier, &player.PartyID, &player.ArenaRank,
		&player.Email, &player.GoogleID, &player.TempLoot, &player.ExploreCount, &player.LastExploreReset,
		&player.ApertureStress, &player.LastCultivateTime,
		&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine, &player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.RefCode, &player.AvatarURL, &player.NeedsOnboarding,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create player: " + err.Error()})
		return
	}

	// Persist starter Gus in the player_gu table (the players.gu_inventory column is gone).
	if err := db.SavePlayerGus(tx, player.ID.String(), starterGu); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to seed starter Gus: " + err.Error()})
		return
	}
	player.GuInventory = starterGu

	// Seed starter materials
	InventorySvc.AddItem(tx, player.ID.String(), "moon_orchid_petal", "materials", 5)
	InventorySvc.AddItem(tx, player.ID.String(), "green_bamboo_wine", "materials", 2)
	InventorySvc.AddItem(tx, player.ID.String(), "jade_stone", "materials", 1)
	// Seed starter recipes
	RecipeSvc.Upsert(tx, &services.RecipeProgress{PlayerID: player.ID.String(), RecipeID: "recipe_moon_raiment", CompletionRate: 100, IsUnlocked: true})
	RecipeSvc.Upsert(tx, &services.RecipeProgress{PlayerID: player.ID.String(), RecipeID: "recipe_white_jade", CompletionRate: 50, IsUnlocked: true})
	RecipeSvc.Upsert(tx, &services.RecipeProgress{PlayerID: player.ID.String(), RecipeID: "recipe_four_flavors_liquor", CompletionRate: 30, IsUnlocked: true})

	if err == nil && len(activeCombatBytes) > 0 && string(activeCombatBytes) != "null" {
		var cs models.CombatState
		if json.Unmarshal(activeCombatBytes, &cs) == nil {
			player.ActiveCombat = &cs
		}
	}

	// 3.5 Handle referral code logic
	trimmedRef := strings.TrimSpace(req.ReferralCode)
	if trimmedRef != "" {
		var referrerID uuid.UUID
		err = tx.QueryRow("SELECT id FROM players WHERE ref_code = $1", trimmedRef).Scan(&referrerID)
		if err == nil {
			// Do not allow self referral
			if referrerID != player.ID {
				clientIP := c.ClientIP()
				userAgent := c.Request.UserAgent()
				var count int
				// Flag as suspicious if the same client IP and device is already referred
				_ = tx.QueryRow("SELECT COUNT(*) FROM referrals WHERE referee_ip = $1 AND referee_device = $2", clientIP, userAgent).Scan(&count)
				isSuspicious := count > 0

				_, err = tx.Exec(`
					INSERT INTO referrals (referrer_id, referee_id, referee_ip, referee_device, is_suspicious)
					VALUES ($1, $2, $3, $4, $5)
				`, referrerID, player.ID, clientIP, userAgent, isSuspicious)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record referral: " + err.Error()})
					return
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save player record: " + err.Error()})
		return
	}

	// 4. Generate token
	token, err := auth.GenerateToken(player.ID, player.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create access token"})
		return
	}
	setAuthCookie(c, token)

	auditlog.Logf(db.DB, &player.ID, "player.register",
		"username=%s email=%s region=%s", req.Username, req.Email, req.StartingRegion)

	c.JSON(http.StatusCreated, CreatePlayerResponse{
		Player: player,
		Token:  token,
	})
}

type CompleteOnboardingRequest struct {
	StartingRegion string `json:"starting_region" binding:"required"`
	AttackGuID     string `json:"attack_gu_id"`
	DefenseGuID    string `json:"defense_gu_id"`
	VitalGuChoice  string `json:"vital_gu_choice"`
}

func CompleteOnboarding(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CompleteOnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validRegions := map[string]bool{
		"Southern_Border": true, "Central_Continent": true,
		"Northern_Plains": true, "Western_Desert": true, "Eastern_Sea": true,
	}
	if !validRegions[req.StartingRegion] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Starting region không hợp lệ"})
		return
	}

	if req.AttackGuID == "" {
		req.AttackGuID = "GU_MOONLIGHT"
	}
	if req.DefenseGuID == "" {
		req.DefenseGuID = "GU_STONE_SKIN"
	}

	validAttackGus := map[string]bool{
		"GU_MOONLIGHT": true, "GU_STREAM": true, "GU_BONE_SPEAR": true,
	}
	validDefenseGus := map[string]bool{
		"GU_GREEN_SILK": true, "GU_STONE_SKIN": true, "GU_IRON_SKIN": true,
		"GU_BEAST_SKIN": true, "GU_FISH_SCALE": true,
	}
	if !validAttackGus[req.AttackGuID] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng tấn công không hợp lệ"})
		return
	}
	if !validDefenseGus[req.DefenseGuID] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng phòng thủ không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

	if !player.NeedsOnboarding {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Onboarding đã hoàn tất"})
		return
	}

	attackGuInstanceID := uuid.New().String()
	defenseGuInstanceID := uuid.New().String()

	isAttackVital := req.VitalGuChoice == "ATTACK"
	isDefenseVital := req.VitalGuChoice == "DEFENSE"

	newGu := models.GuInventory{
		{InstanceID: attackGuInstanceID, GuID: req.AttackGuID, Rank: 1, Hunger: 100, Status: "ACTIVE", Equipped: true, IsVital: isAttackVital},
		{InstanceID: defenseGuInstanceID, GuID: req.DefenseGuID, Rank: 1, Hunger: 100, Status: "ACTIVE", Equipped: true, IsVital: isDefenseVital},
	}
	equippedGus := models.EquippedGusList{attackGuInstanceID, defenseGuInstanceID}

	// Award 20x food of the selected Gus
	gusToProcess := []string{req.AttackGuID, req.DefenseGuID}
	for _, guID := range gusToProcess {
		if guID == "" {
			continue
		}
		spec, ok := models.GetGuAny(guID)
		if !ok {
			continue
		}
		for _, food := range spec.FoodTypes {
			if food == "primeval_stone" {
				player.PrimevalStones += 20
			} else if food == "immortal_stone" {
				player.ImmortalStones += 20.0
			} else {
				// Regular item
				if err := InventorySvc.AddItem(tx, playerID.String(), food, "materials", 20); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add food items: " + err.Error()})
					return
				}
			}
		}
	}

	_, err = tx.Exec(`
		UPDATE players
		SET needs_onboarding = false, current_region = $1, equipped_gus = $2, primeval_stones = $3, immortal_stones = $4
		WHERE id = $5
	`, req.StartingRegion, equippedGus, player.PrimevalStones, player.ImmortalStones, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save onboarding: " + err.Error()})
		return
	}
	if err := db.SavePlayerGus(tx, playerID.String(), newGu); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save starter Gus: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save"})
		return
	}

	player.NeedsOnboarding = false
	player.CurrentRegion = req.StartingRegion
	player.GuInventory = newGu
	player.EquippedGus = equippedGus
	player.MaterialsInventory = loadMaterialsInventory(playerID.String())

	c.JSON(http.StatusOK, player)
}

// LoginRequest represents credentials login payload.
// Email is the canonical login identifier; Username is accepted only as a
// backwards-compatible JSON field for older clients that already send an email
// value under "username".
type LoginRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginPlayer validates cultivator credentials and signs a JWT token
func LoginPlayer(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email và mật khẩu là bắt buộc"})
		return
	}

	loginEmail := strings.ToLower(strings.TrimSpace(req.Email))
	if loginEmail == "" {
		loginEmail = strings.ToLower(strings.TrimSpace(req.Username))
	}
	if loginEmail == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email và mật khẩu là bắt buộc"})
		return
	}
	if !isValidEmail(loginEmail) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng đăng nhập bằng email hợp lệ"})
		return
	}

	var player models.Player
	var passwordHash string

	query := `
		SELECT id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, ` + db.GuInventoryAggregate("players.id") + `, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, last_mana_update, created_at, lifespan, time_speed_multiplier, party_id, arena_rank, COALESCE(email, ''), google_id, temp_loot, explore_count, last_explore_reset, aperture_stress, last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine, COALESCE(bio, ''), COALESCE(fake_rank, 1), COALESCE(fake_sub_rank, 1), COALESCE(fake_vital_gu, ''), COALESCE(is_disguised, false), COALESCE(is_admin, false), COALESCE(avatar_url, '')
		FROM players
		WHERE LOWER(email) = $1
	`
	var activeCombatBytes []byte
	err := db.DB.QueryRow(query, loginEmail).Scan(
		&player.ID, &player.Username, &passwordHash, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones, &player.GuInventory,
		&player.WallProgress, &player.HumanQi, &player.NextTribulationTime, &player.TribulationCount,
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
		&player.EquippedGus, &player.ClearedCells,
		&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries, &activeCombatBytes,
		&player.LastManaUpdate, &player.CreatedAt,
		&player.Lifespan, &player.TimeSpeedMultiplier, &player.PartyID, &player.ArenaRank,
		&player.Email, &player.GoogleID, &player.TempLoot, &player.ExploreCount, &player.LastExploreReset,
		&player.ApertureStress, &player.LastCultivateTime,
		&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine, &player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.AvatarURL,
	)
	if err == nil && len(activeCombatBytes) > 0 && string(activeCombatBytes) != "null" {
		var cs models.CombatState
		if json.Unmarshal(activeCombatBytes, &cs) == nil {
			player.ActiveCombat = &cs
		}
	}

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Thông tin đăng nhập không hợp lệ!"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu: " + err.Error()})
		}
		return
	}

	// Verify hashed password
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// Lazy evaluate mana regen upon login for accurate stats immediately
	tx, err := db.DB.Begin()
	if err == nil {
		defer tx.Rollback()
		if updatedPlayer, err := evaluateManaTx(tx, player.ID); err == nil {
			if err := tx.Commit(); err == nil {
				player = updatedPlayer
			}
		}
	}

	token, err := auth.GenerateToken(player.ID, player.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sign access token"})
		return
	}
	setAuthCookie(c, token)

	auditlog.Logf(db.DB, &player.ID, "player.login",
		"username=%s email=%s", player.Username, loginEmail)

	c.JSON(http.StatusOK, CreatePlayerResponse{
		Player: player,
		Token:  token,
	})
}

// LogoutPlayer clears the auth cookie for the current browser session.
func LogoutPlayer(c *gin.Context) {
	if playerID, exists := middleware.GetPlayerID(c); exists {
		auditlog.Logf(db.DB, &playerID, "player.logout", "")
	}
	clearAuthCookie(c)
	c.Status(http.StatusNoContent)
}

// GetCurrentPlayer returns the authenticated player from the httpOnly cookie session.
func GetCurrentPlayer(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid auth session"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to evaluate player status: " + err.Error()})
		}
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist mana update"})
		return
	}

	player.MaterialsInventory = loadMaterialsInventory(playerID.String())
	c.JSON(http.StatusOK, player)
}

// GetPlayerStatus retrieves a projected player status for UI refresh without persisting lazy state.
func GetPlayerStatus(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid player UUID"})
		return
	}

	// AUTH SECURITY CHECK: Verify token matches the requested resource
	authID, exists := middleware.GetPlayerID(c)
	if !exists || authID != playerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: You do not own this cultivation Aperture"})
		return
	}

	player, err := projectPlayerStatusTx(db.DB, playerID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to project player status: " + err.Error()})
		}
		return
	}

	player.MaterialsInventory = loadMaterialsInventory(playerID.String())
	c.JSON(http.StatusOK, player)
}

// GetPlayerInventoryGus returns the full Gu inventory of the authenticated player.
// Uses a lightweight query against player_gu directly instead of loading the full player row.
func GetPlayerInventoryGus(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	guInv, err := db.LoadPlayerGus(db.DB, authID.String(), false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load Gu inventory: " + err.Error()})
		return
	}

	equippedGus := make([]string, 0)
	for _, gu := range guInv {
		if gu.Equipped && gu.Status == "ACTIVE" {
			equippedGus = append(equippedGus, gu.InstanceID)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"gu_inventory": guInv,
		"equipped_gus": equippedGus,
	})
}

type RechargeWithStoneRequest struct {
	Stones int `json:"stones" binding:"required,min=1"`
}

// RechargeWithStones converts primeval stones into current mana for mortals
func RechargeWithStones(c *gin.Context) {
	var req RechargeWithStoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	// Cổ Tiên (Rank 6+) sử dụng Tiên Nguyên Thạch, không nạp bằng Nguyên Thạch phàm tục
	if player.Rank >= 6 {
		if player.ImmortalStones < float64(req.Stones) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ Tiên Nguyên Thạch."})
			return
		}

		stonesToUse := req.Stones
		manaGain := float64(stonesToUse) / math.Pow(100, float64(player.Rank-6))
		player.CurrentMana += manaGain
		player.ImmortalStones -= float64(stonesToUse)
		if player.CurrentMana > player.MaxMana {
			player.CurrentMana = player.MaxMana
		}

		_, err = tx.Exec(`
			UPDATE players 
			SET current_mana = $1, immortal_stones = $2, last_mana_update = NOW()
			WHERE id = $3
		`, player.CurrentMana, player.ImmortalStones, player.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Tiên Nguyên"})
			return
		}

		_, _ = tx.Exec(`UPDATE immortal_apertures SET immortal_essence = $1 WHERE player_id = $2`, int(player.CurrentMana), player.ID)

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
			return
		}

		responseWithPlayer(c, http.StatusOK, player, gin.H{
			"message": fmt.Sprintf("Luyện hóa thành công %d Tiên Nguyên Thạch thành %d hạt Tiên Nguyên!", stonesToUse, int(manaGain)),
		})
		return
	}

	if player.CurrentMana >= player.MaxMana {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chân nguyên đã đầy, không cần nạp thêm."})
		return
	}

	// --- BẮT ĐẦU CHỈNH SỬA LOGIC MATCH VỚI TRUYỆN ---
	// Mốc cơ bản: 1 viên nguyên thạch hồi 10% Chân nguyên cho Cổ sư Nhất chuyển Sơ giai (Rank 1, SubRank 1)
	baseManaPerStone := 20.0

	// 1. Hệ số nén của Tiểu cảnh giới (Sơ, Trung, Cao, Đỉnh)
	// Nguyên tác: 2 thành Sơ giai luyện thành 1 thành Trung giai -> Hệ số x2 mỗi tiểu cảnh giới.
	subRankMultiplier := math.Pow(2.0, float64(player.SubRank-1))

	// 2. Hệ số nén của Đại cảnh giới (Nhất chuyển lên Nhị chuyển...)
	// Nguyên tác: Chênh lệch chất lượng chân nguyên giữa các Đại cảnh giới cực lớn, lấy mốc x10 lần.
	rankMultiplier := math.Pow(10.0, float64(player.Rank-1))

	// Lượng % chân nguyên thực tế mà 1 viên nguyên thạch hồi được ở Level hiện tại của Player
	manaPerStone := baseManaPerStone / (rankMultiplier * subRankMultiplier)
	// ------------------------------------------------

	neededMana := player.MaxMana - player.CurrentMana

	// Tính số Nguyên Thạch TỐI ĐA cần dùng để sạc đầy Không khiếu
	maxUsableStones := math.Ceil(neededMana / manaPerStone)

	stonesToUse := float64(req.Stones)
	if stonesToUse > maxUsableStones {
		stonesToUse = maxUsableStones
	}

	if stonesToUse > player.PrimevalStones {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch trong túi."})
		return
	}

	// Tính toán số chân nguyên cộng vào
	addedMana := stonesToUse * manaPerStone

	// Tránh việc nạp bị lố MaxMana do làm tròn math.Ceil
	if player.CurrentMana+addedMana > player.MaxMana {
		addedMana = player.MaxMana - player.CurrentMana
		player.CurrentMana = player.MaxMana
	} else {
		player.CurrentMana += addedMana
	}

	player.PrimevalStones -= stonesToUse
	player.LastManaUpdate = time.Now()

	updateQuery := `
		UPDATE players SET current_mana = $1, primeval_stones = $2, last_mana_update = $3
		WHERE id = $4
	`
	_, err = tx.Exec(updateQuery, player.CurrentMana, player.PrimevalStones, player.LastManaUpdate, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save recharge"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"message": fmt.Sprintf("Nạp %.0f nguyên thạch, khôi phục %.4f%% chân nguyên.", stonesToUse, addedMana),
	})
}

// ExploreResponse represents the action of exploring
type ExploreResponse struct {
	Player       models.Player `json:"player"`
	RewardStones float64           `json:"reward_stones"`
	Log          string        `json:"log"`
}

// Explore handles deducting mana, giving player primeval stones, and drops random food materials
func Explore(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid player UUID"})
		return
	}

	// AUTH SECURITY CHECK: Verify token matches the requested resource
	authID, exists := middleware.GetPlayerID(c)
	if !exists || authID != playerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: You do not own this cultivation Aperture"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// 1. Evaluate latest mana status first
	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		}
		return
	}

	// 2. Check energy cost. Mortals spend Chân Nguyên; Immortals spend a smaller Tiên Nguyên travel upkeep.
	energyCost := 10.0
	if isImmortalCultivator(&player) {
		energyCost = 0.5
	}
	if player.CurrentMana < energyCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": insufficientEnergyMessage(&player)})
		return
	}

	// 3. Deduct energy and roll reward stones
	player.CurrentMana -= energyCost
	rewardStones := float64(1 + rand.Intn(3)) // 1 to 3 primeval stones
	player.PrimevalStones += rewardStones
	player.LastManaUpdate = time.Now()

	// 4. Roll random material drop for Phase 2 & 3
	rollMat := rand.Intn(100)
	var droppedMaterial string
	var droppedMaterialName string

	if rollMat < 20 {
		droppedMaterial = "moon_orchid_petal"
		droppedMaterialName = "Moon Orchid Petal (Cánh hoa Nguyệt Lan)"
	} else if rollMat < 35 {
		droppedMaterial = "green_bamboo_wine"
		droppedMaterialName = "Green Bamboo Wine (Rượu Thanh Trúc)"
	} else if rollMat < 50 {
		droppedMaterial = "jade_stone"
		droppedMaterialName = "Jade Stone (Ngọc Thạch)"
	} else if rollMat < 65 {
		droppedMaterial = "boar_meat"
		droppedMaterialName = "Boar Meat (Thịt Lợn Rừng)"
	} else if rollMat < 80 {
		droppedMaterial = "wild_boar_tusk"
		droppedMaterialName = "Wild Boar Tusk (Nanh Lợn Rừng)"
	} else {
		// Roll a random wine type
		wines := []string{"sour_wine", "sweet_wine", "bitter_wine", "spicy_wine"}
		wineNames := []string{"Sour Wine (Rượu Chua)", "Sweet Wine (Rượu Ngọt)", "Bitter Wine (Rượu Đắng)", "Spicy Wine (Rượu Cay)"}
		idx := rand.Intn(len(wines))
		droppedMaterial = wines[idx]
		droppedMaterialName = wineNames[idx]
	}

	inventoryFullMsg := ""

	if droppedMaterial != "" {
		if HasInventorySpace(&player, 1) {
			InventorySvc.AddItem(tx, player.ID.String(), droppedMaterial, "materials", 1)
		} else {
			droppedMaterial = ""
			inventoryFullMsg = " [Hành lý đầy, vật phẩm bị rơi mất!]"
		}
	}

	// Roll 10% chance to capture wild Gu bugs
	rollGu := rand.Intn(100)
	var capturedGuID string
	var capturedGuName string
	if rollGu < 10 { // 10% chance overall
		subRoll := rand.Intn(10) // 0 to 9
		if subRoll < 5 {         // 5% absolute chance
			capturedGuID = "GU_JADE_SKIN"
			capturedGuName = "Ngọc Bì Cổ (Jade Skin Gu)"
		} else if subRoll < 8 { // 3% absolute chance
			capturedGuID = "GU_LIQUOR_WORM"
			capturedGuName = "Tửu Trùng (Liquor Worm)"
		} else { // 2% absolute chance
			capturedGuID = "GU_WHITE_BOAR"
			capturedGuName = "Bạch Thỉ Cổ (White Boar Gu)"
		}
	}

	if capturedGuID != "" {
		if HasInventorySpace(&player, 1) {
			guSpec, _ := models.GetGuAny(capturedGuID)
			newGu := models.GuInstance{
				InstanceID: uuid.New().String(),
				GuID:       capturedGuID,
				Rank:       guSpec.Rank,
				Hunger:     100,
				Status:     "ACTIVE",
				Equipped:   false,
			}
			player.GuInventory = append(player.GuInventory, newGu)
		} else {
			capturedGuID = ""
			inventoryFullMsg = " [Hành lý đầy, Cổ trùng hoang dã trốn thoát!]"
		}
	}

	// Roll 5% chance to find a recipe (Tàn Phương)
	rollRecipe := rand.Intn(100)
	if rollRecipe < 5 {
		// Fetch all recipe IDs from refining_recipes
		var allRecipeIDs []string
		rows, err := tx.Query(`SELECT ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text) FROM refining_recipes`)
		if err == nil {
			for rows.Next() {
				var rID string
				if rows.Scan(&rID) == nil {
					allRecipeIDs = append(allRecipeIDs, rID)
				}
			}
			rows.Close()
		}

		if len(allRecipeIDs) > 0 {
			// Find which ones player hasn't unlocked or isn't 100%
			recipes, _ := RecipeSvc.ListByPlayer(player.ID.String())
			unlockedMap := make(map[string]float64)
			for _, r := range recipes {
				unlockedMap[r.RecipeID] = r.CompletionRate
			}

			var candidates []string
			for _, rID := range allRecipeIDs {
				if rate, ok := unlockedMap[rID]; !ok || rate < 100.0 {
					candidates = append(candidates, rID)
				}
			}

			if len(candidates) > 0 {
				chosenCandidate := candidates[rand.Intn(len(candidates))]
				randomRate := float64(30 + rand.Intn(30)) // 30% to 60%

				// Add or update
				RecipeSvc.Upsert(tx, &services.RecipeProgress{
					PlayerID:       player.ID.String(),
					RecipeID:       chosenCandidate,
					CompletionRate: randomRate,
					IsUnlocked:     true,
				})

				// Find target Gu name
				var targetGuName string
				var targetGuID string
				if tg, _, _, err := ParseRecipeID(chosenCandidate); err == nil {
					targetGuID = tg
				}
				_ = tx.QueryRow(`
					SELECT name FROM gu_dictionary 
					WHERE id = $1
				`, targetGuID).Scan(&targetGuName)

				if targetGuName == "" {
					targetGuName = chosenCandidate
				}

				// Set capturedGuID as non-empty so we can append to logMsg below
				capturedGuName = targetGuName              // reuse variable to communicate name
				capturedGuID = "RECIPE:" + chosenCandidate // mark as recipe
			}
		}
	}

	// 5. Persist changes
	qiGained := AddHumanQi(&player, 3.0)
	updateQuery := `
		UPDATE players
		SET current_mana = $1, primeval_stones = $2, last_mana_update = $3, explored_cells = $4, human_qi = $5
		WHERE id = $6
	`
	if err := db.SavePlayerGus(tx, player.ID.String(), player.GuInventory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update explore progress: " + err.Error()})
		return
	}
	_, err = tx.Exec(updateQuery, player.CurrentMana, player.PrimevalStones, player.LastManaUpdate, player.ExploredCells, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update explore progress: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	// Cultivation logs
	exploreLogs := []string{
		"Đạo hữu khám phá rừng trúc Thanh Mao Sơn và nhặt được %d Nguyên Thạch!",
		"Đạo hữu đánh bại dã trư rừng và thu hoạch %d Nguyên Thạch từ thi thể nó.",
		"Đạo hữu leo lên vách đá gồ ghề và phát hiện %d Nguyên Thạch giấu trong khe đá.",
		"Đạo hữu ngồi bên dòng suối nhỏ yên bình đãi cát tìm thấy %d Nguyên Thạch.",
	}
	selectedLogTemplate := exploreLogs[rand.Intn(len(exploreLogs))]
	logMsg := fmt.Sprintf(selectedLogTemplate, rewardStones)

	if droppedMaterial != "" {
		logMsg = fmt.Sprintf("%s Ngoài ra, đạo hữu còn tìm thấy 1x %s!", logMsg, droppedMaterialName)
	}

	if capturedGuID != "" {
		if strings.HasPrefix(capturedGuID, "RECIPE:") {
			logMsg = fmt.Sprintf("%s Ngoài ra, đạo hữu phát hiện một tờ Tàn Phương của %s!", logMsg, capturedGuName)
		} else {
			logMsg = fmt.Sprintf("%s Và đạo hữu đã thu phục thành công một con %s hoang dã!", logMsg, capturedGuName)
		}
	}

	if inventoryFullMsg != "" {
		logMsg = logMsg + inventoryFullMsg
	}

	logMsg += fmt.Sprintf(" Nhân Khí +%.0f.", qiGained)

	c.JSON(http.StatusOK, ExploreResponse{
		Player:       player,
		RewardStones: rewardStones,
		Log:          logMsg,
	})
}

// FeedRequest represents payload for feeding Gu
type FeedRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

// FeedGu consumes a matching material and resets a Gu instance's hunger status

func getFoodNutrition(foodID string) int {
	if strings.HasPrefix(foodID, "GU_LIFESPAN_") {
		yearsStr := strings.TrimPrefix(foodID, "GU_LIFESPAN_")
		years, err := strconv.Atoi(yearsStr)
		if err == nil {
			return years * 1000
		}
	}
	if spec, ok := models.GetGuAny(foodID); ok {
		return spec.Rank * 1000
	}
	if spec, ok := models.ItemRegistry[foodID]; ok {
		switch spec.Rank {
		case 1:
			return 100
		case 2:
			return 200
		case 3:
			return 500
		case 4:
			return 1000
		case 5:
			return 2000
		case 6:
			return 5000
		case 7:
			return 10000
		case 8:
			return 50000
		case 9:
			return 100000
		default:
			return 100
		}
	}
	return 100
}

func FeedGu(c *gin.Context) {
	var req FeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gu_instance_id is required"})
		return
	}

	// Retrieve auth player ID
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	// 1. Fetch player details with FOR UPDATE lock
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	// 2. Find target Gu instance
	guIndex := -1
	for i, gu := range player.GuInventory {
		if gu.InstanceID == req.GuInstanceID {
			guIndex = i
			break
		}
	}

	if guIndex == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Gu bug not found in your inventory"})
		return
	}

	targetGu := &player.GuInventory[guIndex]
	if targetGu.Status == "DEAD" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This Gu bug is DEAD and cannot be fed"})
		return
	}

	// 3. Find static specs in Dictionary
	specs, found := models.GetGuAny(targetGu.GuID)
	if !found {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gu definition not found in registry"})
		return
	}

	// 4. Loop consuming food until full
	foodOptions := specs.FoodTypes
	foodConsumed := make(map[string]int)
	var feedEventMsg string

	if targetGu.Hunger >= specs.MaxHunger {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng đã no, không cần ăn thêm."})
		return
	}

	for targetGu.Hunger < specs.MaxHunger {
		chosenFoodItem := ""
		consumedGuIndex := -1

		for _, food := range foodOptions {
			if food == "primeval_stone" {
				cost := math.Pow(2, float64(targetGu.Rank))
				if player.PrimevalStones >= cost {
					chosenFoodItem = food
					break
				}
			} else if food == "immortal_stone" {
				cost := float64(targetGu.Rank)
				if player.ImmortalStones >= cost {
					chosenFoodItem = food
					break
				}
			} else {
				qty, _ := InventorySvc.GetQuantity(tx, player.ID.String(), food, "materials")
				if qty > 0 {
					chosenFoodItem = food
					break
				}
			}

			if strings.HasPrefix(food, "GU_") {
				for i, invGu := range player.GuInventory {
					if invGu.GuID == food &&
						invGu.InstanceID != targetGu.InstanceID &&
						invGu.Status != "DEAD" {
						consumedGuIndex = i
						break
					}
				}
				if consumedGuIndex != -1 {
					break
				}
			}
		}

		if chosenFoodItem == "" && consumedGuIndex == -1 {
			break // No more food
		}

		foodID := ""
		if chosenFoodItem != "" {
			if chosenFoodItem != "primeval_stone" && chosenFoodItem != "immortal_stone" {
				InventorySvc.RemoveItem(tx, player.ID.String(), chosenFoodItem, "materials", 1)
			}
			foodID = chosenFoodItem
		} else {
			consumedGu := player.GuInventory[consumedGuIndex]
			foodID = consumedGu.GuID

			// Fire ON_FEED trigger for the consumed Gu
			if cSpec, cOk := models.GetGuAny(consumedGu.GuID); cOk {
				if msg, err := HandleGuEvent(&player, cSpec, consumedGu, "ON_FEED"); err == nil && msg != "" {
					feedEventMsg = msg
				}
			}

			player.GuInventory = append(player.GuInventory[:consumedGuIndex], player.GuInventory[consumedGuIndex+1:]...)

			// Re-find targetGu since slice changed
			guIndex = -1
			for i, gu := range player.GuInventory {
				if gu.InstanceID == req.GuInstanceID {
					guIndex = i
					break
				}
			}
			targetGu = &player.GuInventory[guIndex]
		}

		//some Gu consume immortal_stone and primeval_stone as food, so we need to check if the foodID is a stone and deduct it from the player's inventory
		switch foodID {
		case "immortal_stone":
			cost := float64(targetGu.Rank)
			if player.ImmortalStones < cost {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Tiên Nguyên Thạch để cho Cổ trùng ăn." + fmt.Sprintf(" Cần %d Tiên Nguyên Thạch cho Cổ trùng hạng %d.", int(cost), targetGu.Rank)})
				return
			}
			player.ImmortalStones -= cost
		case "primeval_stone":
			cost := math.Pow(2, float64(targetGu.Rank))
			if player.PrimevalStones < cost {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch để cho Cổ trùng ăn." + fmt.Sprintf(" Cần %.0f Nguyên Thạch cho Cổ trùng hạng %d.", cost, targetGu.Rank)})
				return
			}
			player.PrimevalStones -= cost
		}

		foodConsumed[foodID]++
		nutrition := getFoodNutrition(foodID)
		targetGu.Hunger += nutrition
		if targetGu.Hunger > specs.MaxHunger {
			targetGu.Hunger = specs.MaxHunger
		}
	}

	if len(foodConsumed) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không có thức ăn phù hợp cho %s. Cần một trong: %v", specs.Name, foodOptions)})
		return
	}

	targetGu.Status = "ACTIVE" // Wakes up from sleep (hibernation)
	if targetGu.IsVital {
		targetGu.Equipped = true
	}

	qiGained := AddHumanQi(&player, 2.0)

	// 6. Update database
	if err := db.UpdatePlayerGu(tx, *targetGu); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feeding progress"})
		return
	}
	_, err = tx.Exec(`UPDATE players SET primeval_stones = $1, immortal_stones = $2, hp = $3, max_mana = $4, current_mana = $5, lifespan = $6, human_qi = $7 WHERE id = $8`,
		player.PrimevalStones, player.ImmortalStones, player.HP, player.MaxMana, player.CurrentMana, player.Lifespan, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feeding progress"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	var foodDetails []string
	for fID, qty := range foodConsumed {
		fName := fID
		if itemSpec, ok := models.ItemRegistry[fID]; ok {
			fName = itemSpec.Name
		} else if guSpec, ok := models.GetGuAny(fID); ok {
			fName = guSpec.Name
		}
		foodDetails = append(foodDetails, fmt.Sprintf("%dx %s", qty, fName))
	}
	foodStr := strings.Join(foodDetails, ", ")

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "gu.feed",
		EntityType:     "gu_instance",
		EntityID:       req.GuInstanceID,
		RequestSummary: fmt.Sprintf("Cho Cổ trùng %s (Instance ID: %s) ăn %s", specs.Name, req.GuInstanceID[:8], foodStr),
		Metadata: map[string]interface{}{
			"gu_id":         targetGu.GuID,
			"gu_name":       specs.Name,
			"food_consumed": foodConsumed,
			"new_hunger":    targetGu.Hunger,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	msg := fmt.Sprintf("Cho %s ăn thành công! Đã khôi phục tới %d độ no. Tiêu hao: %s. Nhân Khí +%.0f.", specs.Name, targetGu.Hunger, foodStr, qiGained)
	if feedEventMsg != "" {
		msg += " " + feedEventMsg
	}
	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"message": msg,
	})
}

// FeedGroupRequest represents payload for feeding all hungry Gu bugs of the same type
type FeedGroupRequest struct {
	GuID string `json:"gu_id" binding:"required"`
	Rank *int   `json:"rank,omitempty"`
}

// FeedGroup feeds all hungry Gu bugs of the same type (gu_id) using available food
func FeedGroup(c *gin.Context) {
	var req FeedGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gu_id is required"})
		return
	}

	// Retrieve auth player ID
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	// 1. Fetch player details with FOR UPDATE lock
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	// 2. Find static specs in Dictionary (match both ID and rank when rank provided)
	specs, found := models.GetGuAny(req.GuID)
	if req.Rank != nil {
		if spec, ok := models.GetGu(req.GuID, *req.Rank); ok {
			specs = spec
			found = true
		}
	}
	if !found {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gu definition not found in registry"})
		return
	}

	foodOptions := specs.FoodTypes
	if len(foodOptions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This Gu bug does not require any food or has no food defined"})
		return
	}

	getAvailableFood := func(targetRank int) string {
		for _, food := range foodOptions {
			if food == "primeval_stone" {
				cost := math.Pow(2, float64(targetRank))
				if player.PrimevalStones >= cost {
					return food
				}
			} else if food == "immortal_stone" {
				cost := float64(targetRank)
				if player.ImmortalStones >= cost {
					return food
				}
			} else if !strings.HasPrefix(food, "GU_") {
				qty, _ := InventorySvc.GetQuantity(tx, player.ID.String(), food, "materials")
				if qty > 0 {
					return food
				}
			}
		}
		return ""
	}

	getAvailableGuFood := func(skipInstanceID string) int {
		for _, food := range foodOptions {
			if strings.HasPrefix(food, "GU_") {
				for i, invGu := range player.GuInventory {
					if invGu.GuID == food && invGu.InstanceID != skipInstanceID && invGu.Status != "DEAD" {
						return i
					}
				}
			}
		}
		return -1
	}

	fedCount := 0
	foodConsumed := make(map[string]int)

	// 3. Iterate over the player's Gu inventory and feed hungry ones of matching type
	for i := 0; i < len(player.GuInventory); i++ {
		gu := &player.GuInventory[i]
		if gu.GuID == req.GuID && (req.Rank == nil || gu.Rank == *req.Rank) && gu.Status != "DEAD" && gu.Status != "SEALED" && gu.Hunger < specs.MaxHunger {
			fedThisGu := false
			for gu.Hunger < specs.MaxHunger {
				chosenFood := getAvailableFood(gu.Rank)
				consumedGuIndex := -1
				if chosenFood == "" {
					consumedGuIndex = getAvailableGuFood(gu.InstanceID)
				}

				if chosenFood == "" && consumedGuIndex == -1 {
					break // No more food available
				}

				foodID := ""
				// Consume food
				if chosenFood != "" {
					switch chosenFood {
					case "primeval_stone":
						player.PrimevalStones -= math.Pow(2, float64(gu.Rank))
					case "immortal_stone":
						player.ImmortalStones -= float64(gu.Rank)
					default:
						InventorySvc.RemoveItem(tx, player.ID.String(), chosenFood, "materials", 1)
					}
					foodID = chosenFood
					foodConsumed[chosenFood]++
				} else {
					consumedGu := player.GuInventory[consumedGuIndex]
					foodID = consumedGu.GuID
					foodConsumed[consumedGu.GuID]++
					player.GuInventory = append(player.GuInventory[:consumedGuIndex], player.GuInventory[consumedGuIndex+1:]...)
					if consumedGuIndex <= i {
						i--
					}
					gu = &player.GuInventory[i] // refetch pointer
				}

				nutrition := getFoodNutrition(foodID)
				gu.Hunger += nutrition
				if gu.Hunger > specs.MaxHunger {
					gu.Hunger = specs.MaxHunger
				}
				fedThisGu = true
			}

			if fedThisGu {
				gu.Status = "ACTIVE"
				if gu.IsVital {
					gu.Equipped = true
				}
				fedCount++
			}
		}
	}

	if fedCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có Cổ trùng nào cần cho ăn hoặc không có đủ thức ăn phù hợp"})
		return
	}

	// 4. Update database
	if err := db.SavePlayerGus(tx, player.ID.String(), player.GuInventory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feeding progress"})
		return
	}
	_, err = tx.Exec(`UPDATE players SET primeval_stones = $1, immortal_stones = $2 WHERE id = $3`,
		player.PrimevalStones, player.ImmortalStones, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feeding progress"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	// Build success message
	var foodDetails []string
	for foodID, qty := range foodConsumed {
		foodName := foodID
		if itemSpec, ok := models.ItemRegistry[foodID]; ok {
			foodName = itemSpec.Name
		} else if guSpec, ok := models.GetGuAny(foodID); ok {
			foodName = guSpec.Name
		}
		foodDetails = append(foodDetails, fmt.Sprintf("%dx %s", qty, foodName))
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "gu.feed_group",
		EntityType:     "gu_dictionary",
		EntityID:       req.GuID,
		RequestSummary: fmt.Sprintf("Cho ăn hàng loạt %d con %s. Tiêu hao: %s", fedCount, specs.Name, strings.Join(foodDetails, ", ")),
		Metadata: map[string]interface{}{
			"gu_id":         req.GuID,
			"gu_name":       specs.Name,
			"fed_count":     fedCount,
			"food_consumed": foodConsumed,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"message": fmt.Sprintf("Đã cho ăn thành công %d con %s! Tiêu hao: %s.", fedCount, specs.Name, strings.Join(foodDetails, ", ")),
	})
}

type playerQueryRower interface {
	QueryRow(query string, args ...any) *sql.Row
}

// loadPlayerForStatus delegates to the service layer.
func loadPlayerForStatus(q playerQueryRower, id uuid.UUID, forUpdate bool) (models.Player, error) {
	return PlayerSvc.Load(q, id, forUpdate)
}

func setDefaultPlayerStatusFields(player *models.Player) {
	// Unlocked recipes are now stored in player_recipe_progress table
	if player.Lifespan <= 0 {
		player.Lifespan = 100.0
	}
	if player.TimeSpeedMultiplier <= 0 {
		if player.Rank >= 6 {
			player.TimeSpeedMultiplier = 10.0
		} else {
			player.TimeSpeedMultiplier = 1.0
		}
	}
}

// applyProjectedStatus computes mana regen, HP regen, lifespan decay, and stress decay.
func applyProjectedStatus(player *models.Player, essenceProd int, now time.Time) (bool, bool) {
	poolChanged := normalizeEnergyPool(player)
	stressChanged := false

	var regenRatePerHour float64
	if player.Rank >= 6 {
		subRank := player.SubRank
		if subRank < 1 {
			subRank = 1
		}
		if subRank > 4 {
			subRank = 4
		}
		subRankFactor := 1.0 + float64(subRank-1)*0.15
		regenRatePerHour = (float64(essenceProd) / 120.0) * subRankFactor
	} else {
		regenRatePerHour = energyRegenRatePerHour(player)
	}

	if player.Lifespan <= 0 {
		regenRatePerHour = regenRatePerHour * 0.5
	}

	player.ManaRegenRate = regenRatePerHour

	stats := CalculatePlayerStats(player)
	healingStat := stats["healing"]
	regenHPPerHour := (player.BaseHP * 0.05) + healingStat
	player.HPRegenRate = regenHPPerHour

	secondsPassed := now.Sub(player.LastManaUpdate).Seconds()
	if secondsPassed > 0 {
		subjectiveSeconds := secondsPassed * player.TimeSpeedMultiplier
		earnedMana := (subjectiveSeconds / 3600.0) * regenRatePerHour
		newMana := player.CurrentMana + earnedMana
		if newMana > player.MaxMana {
			newMana = player.MaxMana
		}
		if newMana != player.CurrentMana {
			player.CurrentMana = newMana
			poolChanged = true
		}

		if player.HP > 0 && player.HP < player.BaseHP {
			earnedHP := (subjectiveSeconds / 3600.0) * regenHPPerHour
			newHP := player.HP + earnedHP
			if newHP > player.BaseHP {
				newHP = player.BaseHP
			}
			if newHP != player.HP {
				player.HP = newHP
				poolChanged = true
			}
		}

		yearsPassed := secondsPassed / 36000.0
		player.Lifespan -= yearsPassed
		if player.Lifespan < 0 {
			player.Lifespan = 0
		}
		poolChanged = true
	}

	if player.ApertureStress > 0 && !player.LastCultivateTime.IsZero() {
		elapsedHours := now.Sub(player.LastCultivateTime).Hours()
		if elapsedHours > 0 {
			decay := elapsedHours * 10.0
			newStress := player.ApertureStress - decay
			if newStress < 0 {
				newStress = 0
			}
			if newStress != player.ApertureStress {
				player.ApertureStress = newStress
				poolChanged = true
				stressChanged = true
			}
		}
	}

	return poolChanged, stressChanged
}

func applyImmortalApertureProjection(q playerQueryRower, player *models.Player) (int, bool) {
	essenceProd := 10
	if player.Rank < 6 {
		return essenceProd, false
	}

	var timeRate float64
	if err := q.QueryRow(`SELECT time_rate, essence_production_per_year FROM immortal_apertures WHERE player_id = $1`, player.ID).Scan(&timeRate, &essenceProd); err == nil && timeRate > 0 {
		if player.TimeSpeedMultiplier != timeRate {
			player.TimeSpeedMultiplier = timeRate
			return essenceProd, true
		}
	}

	return essenceProd, false
}

// projectPlayerStatusTx delegates to the service layer.
func projectPlayerStatusTx(q playerQueryRower, id uuid.UUID) (models.Player, error) {
	return PlayerSvc.ProjectStatus(q, id)
}

// evaluateManaTx delegates to the service layer.
func evaluateManaTx(tx *sql.Tx, id uuid.UUID) (models.Player, error) {
	return PlayerSvc.EvaluateAndPersistMana(tx, id)
}

// loadCompletedQuests returns the list of quest IDs the player has completed.
func loadCompletedQuests(playerID string) models.CompletedQuestsList {
	if QuestSvc == nil {
		return nil
	}
	progress, err := QuestSvc.ListCompleted(playerID)
	if err != nil {
		return nil
	}
	result := make(models.CompletedQuestsList, 0, len(progress))
	for _, p := range progress {
		result = append(result, p.QuestID)
	}
	return result
}

// responseWithPlayer sends a JSON response with the player object, ensuring MaterialsInventory and CompletedQuests are populated.
func responseWithPlayer(c *gin.Context, status int, player models.Player, extra ...gin.H) {
	resp := gin.H{"player": player}
	for _, e := range extra {
		for k, v := range e {
			resp[k] = v
		}
	}
	c.JSON(status, resp)
}

// responseWithPlayerData sends a JSON response with pre-loaded player data,
// avoiding redundant DB queries for MaterialsInventory and CompletedQuests.
func responseWithPlayerData(c *gin.Context, status int, player models.Player, materials map[string]int, quests models.CompletedQuestsList, extra ...gin.H) {
	player.MaterialsInventory = materials
	player.CompletedQuests = quests
	resp := gin.H{"player": player}
	for _, e := range extra {
		for k, v := range e {
			resp[k] = v
		}
	}
	c.JSON(status, resp)
}

// loadMaterialsInventory fetches the player's materials inventory as a flat id->quantity map.
func loadMaterialsInventory(playerID string) map[string]int {
	if InventorySvc == nil {
		return nil
	}
	items, err := InventorySvc.ListByPlayer(playerID)
	if err != nil {
		return nil
	}
	mi := make(map[string]int)
	for _, it := range items {
		mi[it.ItemID] += it.Quantity
	}
	return mi
}

// EquipGuRequest defines request to equip/unequip Gu bugs
type EquipGuRequest struct {
	InstanceID string `json:"instance_id" binding:"required"`
	Equip      bool   `json:"equip"`
}

// CalculatePlayerStats computes dynamic stats based on equipped active Gu bugs and passive mods
func CalculatePlayerStats(player *models.Player) map[string]float64 {
	return GetPassiveStats(player).Stats
}

// ── Passive Stats Breakdown ──────────────────────────────────────────────────

// PassiveStatSource describes one source contributing to a passive stat.
type PassiveStatSource struct {
	Source string  `json:"source"` // e.g. "base", "growth", "gu:GU_JADE_SKIN"
	Name   string  `json:"name"`   // human-readable label
	Value  float64 `json:"value"`  // raw contribution before multipliers
	Mult   float64 `json:"mult"`   // path mastery multiplier applied
	Dao    float64 `json:"dao"`    // dao mark multiplier applied
	Final  float64 `json:"final"`  // actual contribution = value * mult * dao
}

// PassiveStatDetail holds the breakdown for one stat key.
type PassiveStatDetail struct {
	Key     string             `json:"key"`
	Total   float64            `json:"total"`
	Sources []PassiveStatSource `json:"sources"`
}

// PassiveStatsResult is the full passive stats response.
type PassiveStatsResult struct {
	Stats   map[string]float64  `json:"stats"`   // flat totals (same as CalculatePlayerStats)
	Details []PassiveStatDetail `json:"details"` // per-stat breakdown
}

// GetPassiveStats computes a detailed breakdown of all passive stats, attributing
// each contribution to its source (base cultivation, growth boosts, equipped Gu).
func GetPassiveStats(player *models.Player) PassiveStatsResult {
	strengthMarks := 0
	if player.DaoMarks != nil {
		strengthMarks = player.DaoMarks["strength"]
	}

	// ── Base stats from Rank / SubRank ──
	baseAttack := 10.0 + float64(player.Rank-1)*10.0 + float64(player.SubRank-1)*2.0 + float64(strengthMarks)/10.0
	baseDefense := 5.0 + float64(player.Rank-1)*5.0 + float64(player.SubRank-1)*1.0
	baseSpeed := 10.0 + float64(player.Rank-1)*2.0 + float64(player.SubRank-1)*0.5

	// Accumulator: key -> []PassiveStatSource
	acc := map[string][]PassiveStatSource{}
	addSrc := func(key, source, name string, value, mult, dao float64) {
		final := value * mult * dao
		if final == 0 && value == 0 {
			return
		}
		acc[key] = append(acc[key], PassiveStatSource{
			Source: source, Name: name, Value: value, Mult: mult, Dao: dao, Final: final,
		})
	}

	// Base sources
	addSrc("attack", "base", "Cơ bản (Cấp bậc)", baseAttack, 1, 1)
	addSrc("defense", "base", "Cơ bản (Cấp bậc)", baseDefense, 1, 1)
	addSrc("speed", "base", "Cơ bản (Cấp bậc)", baseSpeed, 1, 1)
	addSrc("healing", "base", "Cơ bản", 0, 1, 1)
	addSrc("movement", "base", "Cơ bản", 1.0, 1, 1)
	addSrc("investigation", "base", "Cơ bản", 1.0, 1, 1)
	addSrc("storage", "base", "Cơ bản", 100.0, 1, 1)

	// Growth boosts
	if player.GrowthData.Boosts.Attack > 0 {
		addSrc("attack", "growth", "Trưởng thành Cổ", player.GrowthData.Boosts.Attack, 1, 1)
	}
	if player.GrowthData.Boosts.Defense > 0 {
		addSrc("defense", "growth", "Trưởng thành Cổ", player.GrowthData.Boosts.Defense, 1, 1)
	}
	if player.GrowthData.Boosts.Speed > 0 {
		addSrc("speed", "growth", "Trưởng thành Cổ", player.GrowthData.Boosts.Speed, 1, 1)
	}

	// Equipped active Gu passives
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			spec, ok := models.GetGuAny(gu.GuID)
			if !ok || spec.Effects.Passive == nil {
				continue
			}
			passive := spec.Effects.Passive
			mult := getPathMasteryMultiplier(player, spec.Path)

			daoMult := 1.0
			if player.DaoMarks != nil {
				maxDaoMult := 1.0
				for _, path := range spec.Path {
					cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
					if amt, ok := player.DaoMarks[cleanPath]; ok && amt > 0 {
						var pm float64
						if amt <= 10000 {
							pm = 1.0 + float64(amt)/100.0
						} else {
							pm = 1.0 + 100.0 + float64(amt-10000)/500.0
						}
						if pm > maxDaoMult {
							maxDaoMult = pm
						}
					}
				}
				daoMult = maxDaoMult
			}

			srcID := "gu:" + gu.GuID
			if passive.StatBaseAttack > 0 {
				addSrc("attack", srcID, spec.Name, passive.StatBaseAttack, mult, daoMult)
			}
			if passive.StatBaseDefense > 0 {
				addSrc("defense", srcID, spec.Name, passive.StatBaseDefense, mult, daoMult)
			}
			if passive.StatBaseSpeed > 0 {
				addSrc("speed", srcID, spec.Name, passive.StatBaseSpeed, mult, daoMult)
			}
			if passive.StatBaseHP > 0 {
				addSrc("hp", srcID, spec.Name, passive.StatBaseHP, mult, daoMult)
			}
			if passive.DamageMultiplierOnHit > 0 && player.BaseHP > 0 {
				missingHpRate := 1.0 - player.HP/player.BaseHP
				addSrc("damage_multiplier", srcID, spec.Name, missingHpRate*passive.DamageMultiplierOnHit, 1, 1)
			}
			if passive.SoulRefinement > 0 {
				addSrc("soul", srcID, spec.Name, passive.SoulRefinement, 1, 1)
			}
			if passive.StorageCapacity > 0 {
				addSrc("storage", srcID, spec.Name, passive.StorageCapacity, 1, 1)
			}
			if passive.Movement > 0 {
				addSrc("movement", srcID, spec.Name, passive.Movement, 1, 1)
			}
			if passive.Investigation > 0 {
				addSrc("investigation", srcID, spec.Name, passive.Investigation, 1, 1)
			}
			if passive.Hearing > 0 {
				addSrc("hearing", srcID, spec.Name, passive.Hearing, 1, 1)
			}
			if passive.Evasion > 0 {
				addSrc("evasion", srcID, spec.Name, passive.Evasion, 1, 1)
			}
			if passive.CleansingPower > 0 {
				addSrc("healing", srcID, spec.Name, passive.CleansingPower*15.0, 1, 1)
			}
			if passive.Luck > 0 {
				addSrc("luck", srcID, spec.Name, passive.Luck, 1, 1)
			}
		}
	}

	// Build totals and details
	stats := map[string]float64{
		"attack":        0.0,
		"defense":       0.0,
		"speed":         0.0,
		"healing":       0.0,
		"movement":      0.0,
		"investigation": 0.0,
		"storage":       0.0,
	}
	var details []PassiveStatDetail
	for key, srcs := range acc {
		total := 0.0
		for _, s := range srcs {
			total += s.Final
		}
		stats[key] = total
		details = append(details, PassiveStatDetail{Key: key, Total: total, Sources: srcs})
	}

	// Dynamic storage bonus 
	// Only calculated if player rank < 6, as higher ranks have infinite storage
	if player.Rank < 6 {
		storageBonus := math.Floor(stats["attack"] * 0.5)
		stats["storage"] += storageBonus
		for i := range details {
			if details[i].Key == "storage" {
				details[i].Sources = append(details[i].Sources, PassiveStatSource{
					Source: "dynamic", Name: "Lực tấn công (×0.5)", Value: storageBonus, Mult: 1, Dao: 1, Final: storageBonus,
				})
				details[i].Total += storageBonus
				break
			}
		}
	}
	

	// Sort details by key for stable output
	sort.Slice(details, func(i, j int) bool { return details[i].Key < details[j].Key })

	return PassiveStatsResult{Stats: stats, Details: details}
}

// GetPassiveStatsHandler returns a detailed breakdown of all passive stats for the authenticated player.
func GetPassiveStatsHandler(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

	result := GetPassiveStats(&player)
	c.JSON(http.StatusOK, result)
}

// ── Đạo Ngân (Dao Silver) Calculation ───────────────────────────────────────

// DaoNganPathDetail describes the contribution of one dao path to the đạo ngân score.
type DaoNganPathDetail struct {
	Path       string  `json:"path"`       // e.g. "strength", "fire", "sword"
	Amount     int     `json:"amount"`     // raw dao mark count
	Multiplier float64 `json:"multiplier"` // combat multiplier from this path
	Power      float64 `json:"power"`      // effective power contribution
}

// DaoNganResult is the full đạo ngân calculation response.
type DaoNganResult struct {
	TotalDaoNgan    float64             `json:"total_dao_ngan"`    // composite đạo ngân score
	TotalMarks      int                 `json:"total_marks"`       // sum of all dao marks
	UniquePathCount int                 `json:"unique_path_count"` // number of paths with marks
	AverageMultiplier float64           `json:"average_multiplier"` // mean multiplier across paths
	MaxMultiplier   float64             `json:"max_multiplier"`    // highest single-path multiplier
	MaxPath         string              `json:"max_path"`          // path with highest multiplier
	Paths           []DaoNganPathDetail `json:"paths"`             // per-path breakdown
}

// CalculateDaoNgan computes the composite "đạo ngân" (dao silver) score
// from all of a player's dao marks.
//
// Đạo Ngân represents the cumulative cultivation attainment across all dao paths.
// The formula sums each path's effective power:
//
//	power(marks) = marks * multiplier(marks)
//
// where multiplier follows the standard combat formula:
//
//	marks <= 10000: multiplier = 1.0 + marks/100
//	marks  > 10000: multiplier = 101.0 + (marks-10000)/500  (soft cap)
//
// TotalDaoNgan = Σ power(path) for all paths with marks > 0
func CalculateDaoNgan(daoMarks models.DaoMarksMap) DaoNganResult {
	result := DaoNganResult{
		Paths: make([]DaoNganPathDetail, 0),
	}

	if daoMarks == nil {
		return result
	}

	var totalPower float64
	var totalMarks int
	var maxMult float64
	var maxPath string
	var sumMult float64

	for path, amount := range daoMarks {
		if amount <= 0 {
			continue
		}

		var mult float64
		if amount <= 10000 {
			mult = 1.0 + float64(amount)/100.0
		} else {
			mult = 1.0 + 100.0 + float64(amount-10000)/500.0
		}

		power := float64(amount) * mult

		result.Paths = append(result.Paths, DaoNganPathDetail{
			Path:       path,
			Amount:     amount,
			Multiplier: mult,
			Power:      power,
		})

		totalPower += power
		totalMarks += amount
		sumMult += mult
		result.UniquePathCount++

		if mult > maxMult {
			maxMult = mult
			maxPath = path
		}
	}

	// Sort paths by power descending
	sort.Slice(result.Paths, func(i, j int) bool {
		return result.Paths[i].Power > result.Paths[j].Power
	})

	result.TotalDaoNgan = totalPower
	result.TotalMarks = totalMarks
	result.MaxMultiplier = maxMult
	result.MaxPath = maxPath
	if result.UniquePathCount > 0 {
		result.AverageMultiplier = sumMult / float64(result.UniquePathCount)
	}

	return result
}

// GetDaoNganHandler returns the đạo ngân calculation for the authenticated player.
func GetDaoNganHandler(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var daoJSON []byte
	err := db.DB.QueryRowContext(c.Request.Context(),
		`SELECT COALESCE(dao_marks, '{}') FROM players WHERE id = $1`, playerID,
	).Scan(&daoJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load Dao Marks"})
		return
	}

	daoMap := make(models.DaoMarksMap)
	if len(daoJSON) > 0 {
		_ = json.Unmarshal(daoJSON, &daoMap)
	}

	result := CalculateDaoNgan(daoMap)
	c.JSON(http.StatusOK, result)
}

// EquipGu toggles whether a Gu instance is equipped
func EquipGu(c *gin.Context) {
	var req EquipGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player Aperture"})
		return
	}

	targetIdx := -1
	equippedCount := 0
	for i, gu := range player.GuInventory {
		if gu.InstanceID == req.InstanceID {
			targetIdx = i
		}
		if gu.Equipped && gu.Status == "ACTIVE" {
			equippedCount++
		}
	}

	if targetIdx == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Gu bug not found in inventory"})
		return
	}

	targetGu := &player.GuInventory[targetIdx]
	if targetGu.Status == "DEAD" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot equip a DEAD Gu bug"})
		return
	}

	// Vital Gu is always equipped and active — cannot be unequipped
	if targetGu.IsVital && !req.Equip {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bản Mệnh Cổ luôn luôn được trang bị, không thể tháo ra."})
		return
	}

	if req.Equip {
		if targetGu.Equipped {
			responseWithPlayer(c, http.StatusOK, player, gin.H{"message": "Gu bug is already equipped"})
			return
		}
		if equippedCount >= 60 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Aperture capacity full! Cannot equip more than 60 Gu bugs."})
			return
		}

		// Type restriction: block only Consumable Gu from being equipped
		specForEquip, specFound := models.GetGuAny(targetGu.GuID)
		if specFound && specForEquip.Type == "Consumable" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể trang bị Cổ trùng loại Consumable."})
			return
		}

		// Thien Nguyen Bao Lien special cross-rank equip rules
		if specFound && strings.HasPrefix(targetGu.GuID, "GU_HEAVENLY_ESSENCE_LOTUS") {
			if player.Rank < 6 && targetGu.Rank >= 6 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Phàm nhân không thể điều khiển Thiên Nguyên Bảo Hoàng Liên Tiên Cổ!"})
				return
			}
		} else if specFound {
			// All other Gu: standard rank check
			if player.Rank < targetGu.Rank {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không Khiếu Rank %d không thể trang bị Cổ trùng Rank %d!", player.Rank, targetGu.Rank)})
				return
			}
		}

		targetGu.Equipped = true
	} else {
		targetGu.Equipped = false
	}

	player.EquippedGus = []string{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
		}
	}

	guName := targetGu.GuID
	spec, specOk := models.GetGu(targetGu.GuID, targetGu.Rank)
	if specOk {
		guName = spec.Name
	}

	// ── Growth (Growth Effect) ───────────────────────────────────────────
	hasGrowth := specOk && spec.Effects.Growth != nil && spec.Effects.Growth.HoursToMax > 0
	isFullyGrown := false
	if hasGrowth {
		for _, fg := range player.GrowthData.FullyGrown {
			if fg.GuID == targetGu.GuID && fg.Rank == targetGu.Rank {
				isFullyGrown = true
				break
			}
		}
	}

	action := "gu.unequip"
	statusMsg := fmt.Sprintf("Tháo %s thành công!", guName)
	if req.Equip {
		action = "gu.equip"
		statusMsg = fmt.Sprintf("Trang bị %s thành công!", guName)
		if specOk {
			if msg, err := HandleGuEvent(&player, spec, *targetGu, "ON_EQUIP"); err == nil && msg != "" {
				statusMsg += " " + msg
			}
		}
		// Set EquippedAt if this Gu has growth effect and isn't fully grown yet
		if hasGrowth && !isFullyGrown {
			now := time.Now()
			targetGu.EquippedAt = &now
		}
	} else {
		// Unequip — flush pending growth time
		if hasGrowth && targetGu.EquippedAt != nil {
			elapsed := time.Since(*targetGu.EquippedAt).Hours()
			if elapsed > 0 {
				atk, def, hp, spd := ApplyGrowthProgress(targetGu, &spec, elapsed)
				if atk != 0 || def != 0 || hp != 0 || spd != 0 {
					player.GrowthData.Boosts.Attack += atk
					player.GrowthData.Boosts.Defense += def
					player.GrowthData.Boosts.HP += hp
					player.GrowthData.Boosts.Speed += spd
					player.BaseHP += hp
				}
			// Check if fully grown now, add to FullyGrown list
			if targetGu.GrowthHours >= spec.Effects.Growth.HoursToMax {
				player.GrowthData.FullyGrown = append(player.GrowthData.FullyGrown, models.FullyGrownInfo{
					GuID:    targetGu.GuID,
					Rank:    targetGu.Rank,
					Attack:  spec.Effects.Growth.MaxAttack,
					Defense: spec.Effects.Growth.MaxDefense,
					HP:      spec.Effects.Growth.MaxHP,
					Speed:   spec.Effects.Growth.MaxSpeed,
				})
			}
			}
		}
	}

	if err := db.UpdatePlayerGu(tx, *targetGu); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player state"})
		return
	}
	_, err = tx.Exec("UPDATE players SET equipped_gus = $1, primeval_stones = $2, immortal_stones = $3, hp = $4, max_mana = $5, current_mana = $6, lifespan = $7, rank = $8, sub_rank = $9, aptitude_percent = $10, base_hp = $11, growth_data = $12 WHERE id = $13",
		player.EquippedGus, player.PrimevalStones, player.ImmortalStones, player.HP, player.MaxMana, player.CurrentMana, player.Lifespan, player.Rank, player.SubRank, player.AptitudePercent, player.BaseHP, player.GrowthData, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player state"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	auditlog.Logf(db.DB, &authID, action, "gu=%s instance=%s", guName, req.InstanceID)

	responseWithPlayer(c, http.StatusOK, player, gin.H{"message": statusMsg})
}

// UpdateGuStatusRequest defines body to change status of owned Gu instance
type UpdateGuStatusRequest struct {
	InstanceID string `json:"instance_id" binding:"required"`
	Status     string `json:"status" binding:"required"` // "ACTIVE", "SLEEPING", "SEALED"
}

// UpdateGuStatus changes Gu status between ACTIVE, SLEEPING, SEALED
func UpdateGuStatus(c *gin.Context) {
	var req UpdateGuStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Status != "ACTIVE" && req.Status != "SLEEPING" && req.Status != "SEALED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Trạng thái không hợp lệ. Chỉ chấp nhận ACTIVE, SLEEPING hoặc SEALED."})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

	targetIdx := -1
	for idx, inst := range player.GuInventory {
		if inst.InstanceID == req.InstanceID {
			targetIdx = idx
			break
		}
	}

	if targetIdx == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Gu bug not found in inventory"})
		return
	}

	targetGu := &player.GuInventory[targetIdx]
	if targetGu.Status == "DEAD" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể thay đổi trạng thái của Cổ trùng ĐÃ CHẾT."})
		return
	}

	// Vital Gu is always ACTIVE — cannot be put to sleep or sealed
	if targetGu.IsVital && req.Status != "ACTIVE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bản Mệnh Cổ luôn luôn ở trạng thái ACTIVE, không thể thay đổi."})
		return
	}

	// If transitioning away from ACTIVE (e.g. to SLEEPING or SEALED), force unequip if equipped
	if req.Status != "ACTIVE" && targetGu.Equipped {
		targetGu.Equipped = false
	}

	if req.Status == "SEALED" && targetGu.Status != "SEALED" {
		qty, err := InventorySvc.GetQuantity(tx, player.ID.String(), "sealing_stone", "materials")
		if err != nil || qty < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ sealing_stone để phong ấn Cổ trùng."})
			return
		}
		if err := InventorySvc.RemoveItem(tx, player.ID.String(), "sealing_stone", "materials", 1); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to consume sealing_stone"})
			return
		}
	}

	targetGu.Status = req.Status

	// Recalculate equipped_gus list
	player.EquippedGus = []string{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
		}
	}

	if err := db.UpdatePlayerGu(tx, *targetGu); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Gu inventory status"})
		return
	}
	_, err = tx.Exec("UPDATE players SET equipped_gus = $1 WHERE id = $2", player.EquippedGus, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Gu inventory status"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"message": fmt.Sprintf("Cập nhật trạng thái Cổ trùng thành %s thành công!", req.Status),
	})
}

// GetPlayerStatsHandler returns the calculated stats for the current player
func GetPlayerStatsHandler(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player Aperture"})
		return
	}

	stats := CalculatePlayerStats(&player)
	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"stats": stats,
	})
}

// HasInventorySpace checks if the player has enough space in their bag
func HasInventorySpace(player *models.Player, incomingCount int) bool {

	return true
	// stats := CalculatePlayerStats(player)
	// maxSlots := int(stats["storage"])
	// if player.Rank >= 6 {
	// 	maxSlots = 99999999
	// }

	// currentSize := len(player.GuInventory)
	// invItems, _ := InventorySvc.ListByPlayer(player.ID.String())
	// seen := map[string]bool{}
	// for _, inv := range invItems {
	// 	if !seen[inv.ItemID] {
	// 		seen[inv.ItemID] = true
	// 		currentSize++
	// 	}
	// }
	//return (currentSize + incomingCount) <= maxSlots
}

// UseItemRequest defines the request body for using an item
type UseItemRequest struct {
	ItemID string `json:"item_id" binding:"required"`
}

// UseItem consumes a material to restore HP or mana
func UseItem(c *gin.Context) {
	var req UseItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player Aperture"})
		return
	}

	item, ok := models.ItemRegistry[req.ItemID]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm không tồn tại trong ItemRegistry"})
		return
	}

	if item.Type != "Consumable" && item.Type != "SoulResource" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm này không thể sử dụng trực tiếp"})
		return
	}

	// Determine which category the item is in (single query)
	qty, itemCategory, err := InventorySvc.GetQuantityInAnyCategory(tx, player.ID.String(), req.ItemID, "consumables", "materials")
	if err != nil || qty <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có vật phẩm này trong túi đồ"})
		return
	}

	messages := []string{}

	// 1. Validation phase
	for effectKey := range item.Effects {
		if valHandler, ok := ValidateItemEffects[effectKey]; ok {
			_, err := valHandler(tx, &player, item)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
	}

	// 2. Application phase
	for effectKey := range item.Effects {
		if appHandler, ok := ApplyItemEffects[effectKey]; ok {
			msg, err := appHandler(tx, &player, item)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if msg != "" {
				messages = append(messages, msg)
			}

			// If it's a regen item, set Redis cache
			if effectKey == "regen_hp_out_of_combat" {
				redisKey := fmt.Sprintf("regen_buff:%s", player.ID.String())
				cache.SetJSON(redisKey, item.ID, 10*time.Minute)
			}
		}
	}

	if len(messages) == 0 {
		messages = append(messages, fmt.Sprintf("Đạo hữu sử dụng thành công %s.", item.Name))
	}

	// Deduct item from the correct category
	if err := InventorySvc.RemoveItem(tx, player.ID.String(), req.ItemID, itemCategory, 1); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể xóa vật phẩm khỏi túi đồ"})
		return
	}

	// No Gu inventory changes — skip SavePlayerGus
	_, err = tx.Exec(`
		UPDATE players 
		SET current_mana = $1, max_mana = $2, hp = $3, base_hp = $4,
		    dao_marks = $5, primeval_stones = $6, lifespan = $7, human_qi = $8
		WHERE id = $9`,
		player.CurrentMana, player.MaxMana, player.HP, player.BaseHP, player.DaoMarks,
		player.PrimevalStones, player.Lifespan, player.HumanQi,
		player.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player state" + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	player.MaterialsInventory = loadMaterialsInventory(player.ID.String())
	responseWithPlayer(c, http.StatusOK, player, gin.H{"message": strings.Join(messages, " ")})
}

type UseItemsRequest struct {
	ItemIDs []string `json:"item_ids" binding:"required,min=1"`
}

func UseItems(c *gin.Context) {
	var req UseItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.ItemIDs) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ có thể sử dụng tối đa 100 vật phẩm một lúc"})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player Aperture"})
		return
	}

	initialHP := player.HP
	initialMana := player.CurrentMana
	initialLifespan := player.Lifespan
	initialHumanQi := player.HumanQi
	initialStones := player.PrimevalStones
	initialSoul := 0
	if player.DaoMarks != nil {
		initialSoul = player.DaoMarks["soul"]
	}

	messages := []string{}
	regenBuffs := make(map[string]string) // playerID -> itemID

	for _, itemID := range req.ItemIDs {
		item, ok := models.ItemRegistry[itemID]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Vật phẩm %s không tồn tại trong ItemRegistry", itemID)})
			return
		}

		if item.Type != "Consumable" && item.Type != "SoulResource" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Vật phẩm %s này không thể sử dụng trực tiếp", item.Name)})
			return
		}

		// Determine which category the item is in (single query)
		qty, itemCategory, err := InventorySvc.GetQuantityInAnyCategory(tx, player.ID.String(), itemID, "consumables", "materials")
		if err != nil || qty <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không có vật phẩm %s này trong túi đồ", item.Name)})
			return
		}

		// 1. Validation phase
		for effectKey := range item.Effects {
			if valHandler, ok := ValidateItemEffects[effectKey]; ok {
				_, err := valHandler(tx, &player, item)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Vật phẩm %s: %s", item.Name, err.Error())})
					return
				}
			}
		}

		// 2. Application phase
		itemMsg := []string{}
		for effectKey := range item.Effects {
			if appHandler, ok := ApplyItemEffects[effectKey]; ok {
				msg, err := appHandler(tx, &player, item)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Vật phẩm %s: %s", item.Name, err.Error())})
					return
				}
				if msg != "" {
					itemMsg = append(itemMsg, msg)
				}

				// If it's a regen item, set Redis cache later
				if effectKey == "regen_hp_out_of_combat" {
					regenBuffs[player.ID.String()] = item.ID
				}
			}
		}

		if len(itemMsg) == 0 {
			itemMsg = append(itemMsg, fmt.Sprintf("Đạo hữu sử dụng thành công %s.", item.Name))
		}
		messages = append(messages, itemMsg...)

		// Deduct item from the correct category
		if err := InventorySvc.RemoveItem(tx, player.ID.String(), itemID, itemCategory, 1); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không thể xóa vật phẩm %s khỏi túi đồ", item.Name)})
			return
		}
	}

	// No Gu inventory changes — skip SavePlayerGus
	_, err = tx.Exec(`
		UPDATE players 
		SET current_mana = $1, max_mana = $2, hp = $3, base_hp = $4,
		    dao_marks = $5, primeval_stones = $6, lifespan = $7, human_qi = $8
		WHERE id = $9`,
		player.CurrentMana, player.MaxMana, player.HP, player.BaseHP, player.DaoMarks,
		player.PrimevalStones, player.Lifespan, player.HumanQi,
		player.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player state: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	// Apply redis regen buffs
	for pID, itemID := range regenBuffs {
		redisKey := fmt.Sprintf("regen_buff:%s", pID)
		cache.SetJSON(redisKey, itemID, 10*time.Minute)
	}

	// Group/consolidate stats change and messages
	usedCounts := make(map[string]int)
	itemNames := make(map[string]string)
	for _, itemID := range req.ItemIDs {
		if spec, ok := models.ItemRegistry[itemID]; ok {
			usedCounts[itemID]++
			itemNames[itemID] = spec.Name
		}
	}

	var usageDetails []string
	for itemID, count := range usedCounts {
		usageDetails = append(usageDetails, fmt.Sprintf("%dx %s", count, itemNames[itemID]))
	}
	summaryMsg := fmt.Sprintf("Đạo hữu đã sử dụng thành công: %s.", strings.Join(usageDetails, ", "))

	var statChanges []string
	if diffHP := player.HP - initialHP; diffHP > 0 {
		statChanges = append(statChanges, fmt.Sprintf("+%.0f HP", diffHP))
	}
	if diffMana := player.CurrentMana - initialMana; diffMana > 0 {
		statChanges = append(statChanges, fmt.Sprintf("+%.0f Chân Nguyên", diffMana))
	}
	if diffLifespan := player.Lifespan - initialLifespan; diffLifespan > 0 {
		statChanges = append(statChanges, fmt.Sprintf("+%.0f năm Thọ Nguyên", diffLifespan))
	}
	if diffHumanQi := player.HumanQi - initialHumanQi; diffHumanQi > 0 {
		statChanges = append(statChanges, fmt.Sprintf("+%.0f Nhân Khí", diffHumanQi))
	}
	if diffStones := player.PrimevalStones - initialStones; diffStones > 0 {
		statChanges = append(statChanges, fmt.Sprintf("+%.0f Nguyên Thạch", diffStones))
	}
	finalSoul := 0
	if player.DaoMarks != nil {
		finalSoul = player.DaoMarks["soul"]
	}
	if diffSoul := finalSoul - initialSoul; diffSoul > 0 {
		statChanges = append(statChanges, fmt.Sprintf("+%d Hồn Lực", diffSoul))
	}

	if len(statChanges) > 0 {
		summaryMsg += fmt.Sprintf(" Kết quả: %s.", strings.Join(statChanges, ", "))
	}

	var specialMessages []string
	for _, m := range messages {
		isStandard := false
		lowerM := strings.ToLower(m)
		if strings.Contains(lowerM, "hồi phục") && strings.Contains(lowerM, "hp") {
			isStandard = true
		} else if strings.Contains(lowerM, "phục hồi") && strings.Contains(lowerM, "chân nguyên") {
			isStandard = true
		} else if strings.Contains(lowerM, "thọ nguyên") {
			isStandard = true
		} else if strings.Contains(lowerM, "nhân khí") {
			isStandard = true
		} else if strings.Contains(lowerM, "hồn lực") || strings.Contains(lowerM, "hồn phách") {
			isStandard = true
		} else if strings.Contains(lowerM, "tiên khiếu") && strings.Contains(lowerM, "khôi phục") {
			isStandard = true
		} else if strings.Contains(lowerM, "sử dụng thành công") {
			isStandard = true
		}

		if !isStandard && m != "" {
			specialMessages = append(specialMessages, m)
		}
	}

	finalMessage := summaryMsg
	if len(specialMessages) > 0 {
		aggregatedSpecial := aggregateChestSpecialMessages(specialMessages)
		finalMessage += "\n" + strings.Join(aggregatedSpecial, "\n")
	}

	player.MaterialsInventory = loadMaterialsInventory(player.ID.String())
	responseWithPlayer(c, http.StatusOK, player, gin.H{"message": finalMessage})
}

// SetVitalGuRequest represents the payload to select a vital Gu
type SetVitalGuRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

// SetVitalGu sets a selected Gu as the player's Vital Gu
func SetVitalGu(c *gin.Context) {
	var req SetVitalGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gu_instance_id is required"})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	targetIndex := -1
	for idx, gu := range player.GuInventory {
		if gu.InstanceID == req.GuInstanceID {
			targetIndex = idx
			break
		}
	}

	if targetIndex == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng trong túi đồ"})
		return
	}

	targetGu := &player.GuInventory[targetIndex]
	if targetGu.Status == "DEAD" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể thiết lập Cổ trùng ĐÃ CHẾT làm Bản Mệnh Cổ"})
		return
	}

	if targetGu.Rank > player.Rank {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không thể thiết lập Cổ trùng Rank %d làm Bản Mệnh Cổ do vượt quá tu vi hiện tại của đạo hữu (Rank %d)!", targetGu.Rank, player.Rank)})
		return
	}

	if targetGu.IsVital {
		responseWithPlayer(c, http.StatusOK, player, gin.H{
			"message": "Cổ trùng này đã là Bản Mệnh Cổ của đạo hữu rồi.",
		})
		return
	}

	// Check if they already have an existing Vital Gu
	hadVital := false
	for _, gu := range player.GuInventory {
		if gu.IsVital {
			hadVital = true
			break
		}
	}

	backlashMessage := ""
	if hadVital {
		// Backlash: cost 50 HP
		player.HP -= 50.0
		backlashMessage = " Đạo hữu bị phản phệ mất 50 HP do thay đổi Bản Mệnh Cổ!"
		if player.HP <= 0 {
			player.HP = 0
			player.ActiveCombat = nil // reset combat
			backlashMessage += " Đạo hữu khí huyết cạn kiệt, bất tĩnh nhân sự!"
		}
	}

	// Update is_vital flags; vital Gu is always ACTIVE and equipped
	for idx := range player.GuInventory {
		if idx == targetIndex {
			player.GuInventory[idx].IsVital = true
			player.GuInventory[idx].Status = "ACTIVE"
			player.GuInventory[idx].Equipped = true
		} else {
			player.GuInventory[idx].IsVital = false
		}
	}

	// Recalculate equipped_gus
	player.EquippedGus = []string{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
		}
	}

	if err := db.SavePlayerGus(tx, player.ID.String(), player.GuInventory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player vital Gu status"})
		return
	}
	_, err = tx.Exec(`
		UPDATE players 
		SET hp = $1, equipped_gus = $2, active_combat = $3
		WHERE id = $4`,
		player.HP, player.EquippedGus, player.ActiveCombat, player.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player vital Gu status"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "gu.set_vital", EntityType: "gu_instance",
		EntityID:       req.GuInstanceID,
		RequestSummary: fmt.Sprintf("Thiết lập %s thành Bản Mệnh Cổ | hadVital=%v", targetGu.GuID, hadVital),
		IPAddress:      c.ClientIP(), Success: true,
	})
	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"message": fmt.Sprintf("Thiết lập %s thành Bản Mệnh Cổ thành công!%s", targetGu.GuID, backlashMessage),
	})
}

// DropGuRequest removes an owned Gu from inventory intentionally.
type DropGuRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

func DropGu(c *gin.Context) {
	var req DropGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gu_instance_id is required"})
		return
	}
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	idx := -1
	for i, gu := range player.GuInventory {
		if gu.InstanceID == req.GuInstanceID {
			idx = i
			break
		}
	}
	if idx == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng trong túi đồ"})
		return
	}
	if player.GuInventory[idx].IsVital {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể vứt Bản Mệnh Cổ"})
		return
	}

	newInv := models.GuInventory{}
	for i, gu := range player.GuInventory {
		if i != idx {
			newInv = append(newInv, gu)
		}
	}
	player.GuInventory = newInv
	player.EquippedGus = []string{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
		}
	}
	if err := db.RemovePlayerGu(tx, player.GuInventory[idx].InstanceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to drop Gu"})
		return
	}
	_, err = tx.Exec(`UPDATE players SET equipped_gus = $1 WHERE id = $2`, player.EquippedGus, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to drop Gu"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "gu.drop", EntityType: "gu_instance",
		EntityID: req.GuInstanceID, RequestSummary: "Vứt bỏ Cổ trùng khỏi túi đồ",
		IPAddress: c.ClientIP(), Success: true,
	})
	responseWithPlayer(c, http.StatusOK, player, gin.H{"message": "Đã vứt bỏ Cổ trùng khỏi túi đồ."})
}

// DropGusRequest represents request payload to drop multiple Gu instances.
type DropGusRequest struct {
	GuInstanceIDs []string `json:"gu_instance_ids" binding:"required"`
}

// DropGus removes multiple owned Gus from inventory.
func DropGus(c *gin.Context) {
	var req DropGusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gu_instance_ids is required"})
		return
	}
	if len(req.GuInstanceIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Danh sách gu_instance_ids trống"})
		return
	}
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	dropMap := make(map[string]bool)
	for _, id := range req.GuInstanceIDs {
		dropMap[id] = true
	}

	foundCount := 0
	for _, gu := range player.GuInventory {
		if dropMap[gu.InstanceID] {
			foundCount++
			if gu.IsVital {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không thể vứt Bản Mệnh Cổ (%s)", gu.InstanceID)})
				return
			}
		}
	}

	if foundCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng nào trong danh sách yêu cầu"})
		return
	}

	newInv := models.GuInventory{}
	for _, gu := range player.GuInventory {
		if !dropMap[gu.InstanceID] {
			newInv = append(newInv, gu)
		}
	}
	player.GuInventory = newInv
	player.EquippedGus = []string{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
		}
	}
	if err := db.SavePlayerGus(tx, player.ID.String(), player.GuInventory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to drop Gus"})
		return
	}
	_, err = tx.Exec(`UPDATE players SET equipped_gus = $1 WHERE id = $2`, player.EquippedGus, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to drop Gus"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "gu.drop_batch", EntityType: "gu_instance",
		RequestSummary: fmt.Sprintf("Vứt bỏ hàng loạt %d Cổ trùng khỏi túi đồ", len(req.GuInstanceIDs)),
		IPAddress:      c.ClientIP(), Success: true,
	})
	responseWithPlayer(c, http.StatusOK, player, gin.H{"message": fmt.Sprintf("Đã vứt bỏ %d Cổ trùng khỏi túi đồ.", foundCount)})
}

// UseGuRequest represents request payload to use a consumable Gu
type UseGuRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

// UseConsumableGu activates a consumable Gu bug's one-time effect and removes it
func UseConsumableGu(c *gin.Context) {
	var req UseGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gu_instance_id is required"})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	targetIdx := -1
	for idx, gu := range player.GuInventory {
		if gu.InstanceID == req.GuInstanceID {
			targetIdx = idx
			break
		}
	}

	if targetIdx == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng này trong túi đồ"})
		return
	}

	targetGu := &player.GuInventory[targetIdx]
	specs, found := models.GetGuAny(targetGu.GuID)
	if !found {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy thông tin Cổ trùng trong thư viện"})
		return
	}

	if specs.Type != "Consumable" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng này không thuộc loại Tiêu Hao, không thể sử dụng trực tiếp!"})
		return
	}

	if err := ValidateGuUsable(&player, targetGu, &specs, false); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cost := calculateEnergyCost(&player, &specs)
	player.CurrentMana -= cost

	var logs []string
	if specs.Effects.Other != nil {
		for _, mechanic := range specs.Effects.Other {
			if mechanic.EventTrigger == "ON_CONSUME" {
				msg, err := HandleConsumeEvent(&player, *targetGu, specs, mechanic)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				if msg != "" {
					logs = append(logs, msg)
				}
			}
		}
	}

	if len(logs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hiệu ứng của Cổ trùng tiêu hao này chưa được định nghĩa hoặc không kích hoạt thành công!"})
		return
	}

	logMsg := strings.Join(logs, " ")

	qiGained := AddHumanQi(&player, 3.0)

	// Delete from inventory
	RemoveGuFromInventory(&player, targetGu.InstanceID)

	if err := db.RemovePlayerGu(tx, targetGu.InstanceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player state after using Gu"})
		fmt.Println("Failed to update player state after using Gu:", err)
		return
	}
	_, err = tx.Exec(`
		UPDATE players 
		SET sub_rank = $1, rank = $2, primeval_stones = $3, max_mana = $4, current_mana = $5, equipped_gus = $6, dao_marks = $7, wall_progress = $8, hp = $9, lifespan = $10, aptitude_percent = $11, immortal_stones = $12, aperture_stress = $13, human_qi = $14, max_soul = $15
		WHERE id = $16`,
		player.SubRank, player.Rank, player.PrimevalStones, player.MaxMana, player.CurrentMana, player.EquippedGus, player.DaoMarks, player.WallProgress, player.HP, player.Lifespan, player.AptitudePercent, player.ImmortalStones, player.ApertureStress, player.HumanQi, player.MaxSoul, player.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player state after using Gu"})
		fmt.Println("Failed to update player state after using Gu:", err)
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "gu.use_consumable", EntityType: "gu_instance",
		EntityID: req.GuInstanceID, RequestSummary: logMsg,
		IPAddress: c.ClientIP(), Success: true,
	})
	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"message": logMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
	})
}

// UseSupportGuRequest represents request payload to use a support Gu out of combat
type UseSupportGuRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

// UseSupportGu activates an out-of-combat support Gu bug's effect
func UseSupportGu(c *gin.Context) {
	var req UseSupportGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gu_instance_id is required"})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	targetIdx := -1
	for idx, gu := range player.GuInventory {
		if gu.InstanceID == req.GuInstanceID {
			targetIdx = idx
			break
		}
	}

	if targetIdx == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng này trong túi đồ"})
		return
	}

	targetGu := &player.GuInventory[targetIdx]
	specs, found := models.GetGuAny(targetGu.GuID)
	if !found {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy thông tin Cổ trùng trong thư viện"})
		return
	}

	if specs.Type != "Support" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng này không thuộc loại Hỗ Trợ, không thể sử dụng trực tiếp ngoài combat!"})
		return
	}

	if err := ValidateGuUsable(&player, targetGu, &specs, false); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cost := calculateEnergyCost(&player, &specs)
	player.CurrentMana -= cost

	var logs []string
	if specs.Effects.Other != nil {
		for _, mechanic := range specs.Effects.Other {
			if mechanic.EventTrigger == "ON_CONSUME" {
				msg, err := HandleConsumeEvent(&player, *targetGu, specs, mechanic)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				if msg != "" {
					logs = append(logs, msg)
				}
			}
		}
	}

	if len(logs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng hỗ trợ này không có hiệu ứng ngoài chiến đấu!"})
		return
	}

	qiGained := AddHumanQi(&player, float64(3))

	logMsg := strings.Join(logs, " ")
	// Save to DB — support Gu stays in inventory, no need to rewrite inventory
	_, err = tx.Exec(`
		UPDATE players 
		SET max_mana = $1, current_mana = $2, aptitude_percent = $3, hp = $4, aperture_stress = $5, human_qi = $6
		WHERE id = $7`,
		player.MaxMana, player.CurrentMana, player.AptitudePercent, player.HP, player.ApertureStress, player.HumanQi,
		player.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player state after using support Gu"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"message": logMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
	})
}

// Helper function to get rank name in Vietnamese
func getRankName(rank int) string {
	names := map[int]string{
		1: "Nhất Chuyển",
		2: "Nhị Chuyển",
		3: "Tam Chuyển",
		4: "Tứ Chuyển",
		5: "Ngũ Chuyển",
		6: "Lục Chuyển Cổ Tiên",
		7: "Thất Chuyển Cổ Tiên",
		8: "Bát Chuyển Cổ Tiên",
		9: "Cửu Chuyển Cổ Tôn",
	}
	if name, ok := names[rank]; ok {
		return name
	}
	return fmt.Sprintf("%d Chuyển", rank)
}

// UpdateProfileSettingsRequest represents settings data to update
type UpdateProfileSettingsRequest struct {
	Username    string `json:"username"`
	Bio         string `json:"bio"`
	NewPassword string `json:"new_password"`
	FakeRank    int    `json:"fake_rank"`
	FakeSubRank int    `json:"fake_sub_rank"`
	FakeVitalGu string `json:"fake_vital_gu"`
	IsDisguised bool   `json:"is_disguised"`
}

// UpdateProfileSettings updates the current user's profile settings (email, bio, password, fake profile)
func UpdateProfileSettings(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req UpdateProfileSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	// Validate Disguise requirements if player wants to activate or configure disguise
	hasConcealment := false
	hasAttitude := false
	hasFormChange := false
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			switch gu.GuID {
			case "GU_SHADOW", "GU_SHADOW_STEPS", "GU_STEALTH_SCALES":
				hasConcealment = true
			case "GU_CHANGE_FORM":
				hasConcealment = true
				hasFormChange = true
			case "GU_ATTITUDE":
				hasConcealment = true
				hasAttitude = true
			}
		}
	}

	if req.IsDisguised {
		if !hasConcealment {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không trang bị cổ trùng ngụy trang nào hoạt động! Cần trang bị các cổ trùng như Ẩn Lân Cổ, Biến Hóa Cổ, Thái Độ Cổ..."})
			return
		}

		if req.FakeRank != player.Rank {
			if !hasAttitude {
				// Standard stealth only allows hiding/lowering rank ("giả yếu")
				if req.FakeRank > player.Rank || req.FakeRank < 1 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng hiện tại chỉ hỗ trợ giấu khí tức làm yếu đi, không thể giả làm cường giả mạnh hơn tu vi bản thân! Chỉ có Thái Độ Cổ mới đạt tới mức ngụy trang vô ngại."})
					return
				}
			} else {
				if req.FakeRank < 1 || req.FakeRank > 9 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Tu vi giả chỉ có thể từ Nhất Chuyển (1) đến Cửu Chuyển (9)!"})
					return
				}
			}
		}

		if req.FakeVitalGu != "" {
			if !hasAttitude && !hasFormChange {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Cần trang bị Biến Hóa Cổ hoặc Thái Độ Cổ để tạo dựng Bản Mệnh Cổ giả!"})
				return
			}
			// Optional: verify they own this Gu or it is a valid Gu ID in registry
			found := false
			for _, gu := range player.GuInventory {
				if gu.GuID == req.FakeVitalGu {
					found = true
					break
				}
			}
			if req.FakeVitalGu != "GU_SHADOW" && !found {
				// Allow standard fakes or anything they actually own
				c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không sở hữu cổ trùng " + req.FakeVitalGu + " để thiết lập ngụy trang!"})
				return
			}
		}
	}

	username := strings.TrimSpace(req.Username)
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hiệu không được để trống!"})
		return
	}

	if len(username) < 3 || len(username) > 30 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hiệu phải từ 3 đến 30 ký tự!"})
		return
	}

	for _, r := range username {
		if !unicode.IsLetter(r) && !unicode.IsSpace(r) && !unicode.IsDigit(r) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hiệu chỉ được chứa chữ cái, số và khoảng trắng, không chứa ký tự đặc biệt!"})
			return
		}
	}

	if username != player.Username {
		var exists bool
		err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM players WHERE username = $1 AND id <> $2)`, username, player.ID).Scan(&exists)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		if exists {
			c.JSON(http.StatusConflict, gin.H{"error": "Đạo hiệu này đã được sử dụng bởi cổ sư khác!"})
			return
		}
		player.Username = username
	}

	player.Bio = req.Bio
	player.FakeRank = req.FakeRank
	player.FakeSubRank = req.FakeSubRank
	player.FakeVitalGu = req.FakeVitalGu
	player.IsDisguised = req.IsDisguised

	if req.NewPassword != "" {
		if len(req.NewPassword) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mật khẩu mới phải có ít nhất 6 ký tự"})
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi mã hóa mật khẩu"})
			return
		}
		_, err = tx.Exec(`UPDATE players SET username = $1, bio = $2, fake_rank = $3, fake_sub_rank = $4, fake_vital_gu = $5, is_disguised = $6, password_hash = $7 WHERE id = $8`, player.Username, player.Bio, player.FakeRank, player.FakeSubRank, player.FakeVitalGu, player.IsDisguised, string(hash), player.ID)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Đạo hiệu đã được sử dụng bởi cổ sư khác!"})
			return
		}
	} else {
		_, err = tx.Exec(`UPDATE players SET username = $1, bio = $2, fake_rank = $3, fake_sub_rank = $4, fake_vital_gu = $5, is_disguised = $6 WHERE id = $7`, player.Username, player.Bio, player.FakeRank, player.FakeSubRank, player.FakeVitalGu, player.IsDisguised, player.ID)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Đạo hiệu đã được sử dụng bởi cổ sư khác!"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save profile settings"})
		return
	}

	// Update token cookie to keep session claim synchronized with the new username
	newToken, err := auth.GenerateToken(player.ID, player.Username)
	if err == nil {
		setAuthCookie(c, newToken)
	}

	responseWithPlayer(c, http.StatusOK, player, gin.H{"message": "Cập nhật hồ sơ thành công!"})
}

func getClientIDFromGoogleJSON() string {
	candidates := []string{
		"google.json",
		"backend/google.json",
		"../google.json",
		"../../google.json",
	}
	var data []byte
	var err error
	for _, p := range candidates {
		data, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if len(data) == 0 {
		return ""
	}
	var config struct {
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return ""
	}
	return config.ClientID
}

// GoogleLogin redirects user to Google OAuth2 consent page
func GoogleLogin(c *gin.Context) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		clientID = getClientIDFromGoogleJSON()
	}
	if clientID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Google Client ID is not configured"})
		return
	}

	redirectURI := os.Getenv("GOOGLE_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = "http://localhost:8080/api/auth/google/callback"
	}

	stateToken := make([]byte, 16)
	if _, err := crypto_rand.Read(stateToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate security state"})
		return
	}
	stateStr := hex.EncodeToString(stateToken)

	// Set state cookie valid for 5 minutes
	c.SetCookie("google_oauth_state", stateStr, 300, "/api/auth/google", "", authCookieSecure(c, false), true)

	refCode := c.Query("ref")
	state := stateStr + ":" + refCode
	googleURL := fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid+profile+email&state=%s",
		clientID, url.QueryEscape(redirectURI), state)

	c.Redirect(http.StatusTemporaryRedirect, googleURL)
}

// GoogleCallback handles authorization callback from Google
func GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization code not found"})
		return
	}

	state := c.Query("state")
	if state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Security state parameter not found"})
		return
	}

	stateCookie, err := c.Cookie("google_oauth_state")
	if err != nil || stateCookie == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Security state cookie not found or expired"})
		return
	}

	// Xóa cookie ngay lập tức
	c.SetCookie("google_oauth_state", "", -1, "/api/auth/google", "", authCookieSecure(c, false), true)

	parts := strings.SplitN(state, ":", 2)
	if len(parts) == 0 || parts[0] != stateCookie {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid security state mismatch"})
		return
	}

	var referralCode string
	if len(parts) == 2 {
		referralCode = parts[1]
	}

	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		clientID = getClientIDFromGoogleJSON()
	}
	if clientID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Google Client ID is not configured"})
		return
	}

	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURI := os.Getenv("GOOGLE_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = "http://localhost:8080/api/auth/google/callback"
	}

	// Exchange authorization code for token
	tokenURL := "https://oauth2.googleapis.com/token"
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange authorization code: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errData map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errData)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Google token exchange failed", "details": errData})
		return
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse token response"})
		return
	}

	// Fetch user profile using access token
	userinfoURL := "https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + tokenResp.AccessToken
	userinfoResp, err := http.Get(userinfoURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info: " + err.Error()})
		return
	}
	defer userinfoResp.Body.Close()

	if userinfoResp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to retrieve user profile from Google"})
		return
	}

	var googleUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(userinfoResp.Body).Decode(&googleUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user info"})
		return
	}

	if googleUser.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Google account is missing email details"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var player models.Player
	var hasPlayer bool

	// 1. Try to search by google_id
	err = tx.QueryRow(`
		SELECT id, username, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, `+db.GuInventoryAggregate("players.id")+`, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, last_mana_update, created_at, lifespan, time_speed_multiplier, email, google_id, temp_loot, explore_count, last_explore_reset, aperture_stress, last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine, bio, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, is_admin, COALESCE(ref_code, ''), COALESCE(needs_onboarding, false)
		FROM players
		WHERE google_id = $1
	`, googleUser.ID).Scan(
		&player.ID, &player.Username, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones, &player.GuInventory,
		&player.WallProgress, &player.HumanQi, &player.NextTribulationTime, &player.TribulationCount,
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
		&player.EquippedGus, &player.ClearedCells,
		&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries,
		&player.LastManaUpdate, &player.CreatedAt,
		&player.Lifespan, &player.TimeSpeedMultiplier,
		&player.Email, &player.GoogleID, &player.TempLoot, &player.ExploreCount, &player.LastExploreReset,
		&player.ApertureStress, &player.LastCultivateTime,
		&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine, &player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.RefCode, &player.NeedsOnboarding,
	)

	if err == nil {
		hasPlayer = true
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking google_id: " + err.Error()})
		return
	}

	if !hasPlayer {
		// 2. Try to search by email (Merge Account!)
		err = tx.QueryRow(`
			SELECT id, username, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, `+db.GuInventoryAggregate("players.id")+`, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, last_mana_update, created_at, lifespan, time_speed_multiplier, email, google_id, temp_loot, explore_count, last_explore_reset, aperture_stress, last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine, bio, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, is_admin, COALESCE(ref_code, ''), COALESCE(needs_onboarding, false)
			FROM players
			WHERE email = $1
		`, googleUser.Email).Scan(
			&player.ID, &player.Username, &player.Rank, &player.SubRank,
			&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
			&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones, &player.GuInventory,
			&player.WallProgress, &player.HumanQi, &player.NextTribulationTime, &player.TribulationCount,
			&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
			&player.EquippedGus, &player.ClearedCells,
			&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries,
			&player.LastManaUpdate, &player.CreatedAt,
			&player.Lifespan, &player.TimeSpeedMultiplier,
			&player.Email, &player.GoogleID, &player.TempLoot, &player.ExploreCount, &player.LastExploreReset,
			&player.ApertureStress, &player.LastCultivateTime,
			&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine, &player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.RefCode, &player.NeedsOnboarding,
		)

		if err == nil {
			// Merge Account: update players table to link Google account ID
			_, err = tx.Exec("UPDATE players SET google_id = $1 WHERE id = $2", googleUser.ID, player.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to merge account: " + err.Error()})
				return
			}
			gID := googleUser.ID
			player.GoogleID = &gID
			hasPlayer = true
		} else if err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking email: " + err.Error()})
			return
		}
	}

	if !hasPlayer {
		// 3. Create a brand new player
		username := googleUser.Name
		if username == "" {
			parts := strings.Split(googleUser.Email, "@")
			username = parts[0]
		}
		username = strings.TrimSpace(username)
		if len(username) < 3 {
			username = "cultivator_" + strings.ToLower(uuid.New().String()[:6])
		} else if len(username) > 30 {
			username = username[:30]
		}

		// Check duplicate username and resolve
		var exists bool
		err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM players WHERE username = $1)", username).Scan(&exists)
		if err == nil && exists {
			username = fmt.Sprintf("%s_%d", username, rand.Intn(9000)+1000)
		}

		var prod_env = os.Getenv("PROD_ENV") == "test"
		var roll int
		if prod_env {
			roll = rand.Intn(30) // 0-29
		} else {
			roll = rand.Intn(100) // 0-99
		}
		var aptitudeType string
		var aptitudePercent int

		if roll < 10 {
			aptitudeType = "Giáp"
			aptitudePercent = 80 + rand.Intn(20) // 80 - 99
		} else if roll < 30 {
			aptitudeType = "Ất"
			aptitudePercent = 60 + rand.Intn(20) // 60 - 79
		} else if roll < 60 {
			aptitudeType = "Bính"
			aptitudePercent = 40 + rand.Intn(20) // 40 - 59
		} else {
			aptitudeType = "Đinh"
			aptitudePercent = 20 + rand.Intn(20) // 20 - 39
		}

		aptitudeGrade := 4
		switch aptitudeType {
		case "Giáp":
			aptitudeGrade = 1
		case "Ất":
			aptitudeGrade = 2
		case "Bính":
			aptitudeGrade = 3
		case "Đinh":
			aptitudeGrade = 4
		}

		maxMana := float64(aptitudePercent)
		currentMana := maxMana

		attackGuInstanceID := uuid.New().String()
		defenseGuInstanceID := uuid.New().String()

		starterGu := models.GuInventory{
			{
				InstanceID: attackGuInstanceID,
				GuID:       "GU_MOONLIGHT",
				Rank:       1,
				Hunger:     100,
				Status:     "ACTIVE",
				Equipped:   true,
				IsVital:    true,
			},
			{
				InstanceID: defenseGuInstanceID,
				GuID:       "GU_STONE_SKIN",
				Rank:       1,
				Hunger:     100,
				Status:     "ACTIVE",
				Equipped:   true,
				IsVital:    false,
			},
		}
		equippedGus := models.EquippedGusList{attackGuInstanceID, defenseGuInstanceID}

		// Generate unique referral code
		refCode := "REF-" + strings.ToUpper(uuid.New().String()[:8])

		queryInsert := `
			INSERT INTO players (username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, lifespan, time_speed_multiplier, email, google_id, temp_loot, explore_count, last_explore_reset, soul_hp, max_soul, soul_tier, needs_soul_refine, bio, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, ref_code, needs_onboarding)
			VALUES ($1, '', 1, 1, $2, $3, $4, $5, $6, 20, 0, 0.0, 0.0, NULL, 0, 100.0, 100.0, '[]'::jsonb, 50, 50, '["50,50"]'::jsonb, $10, '{}'::jsonb, 'Southern_Border', '{"strength": 0}'::jsonb, '{}'::jsonb, NULL, 100.0, 1.0, $7, $8, '{"gu_inventory": [], "materials_inventory": {}, "primeval_stones": 0}'::jsonb, 0, NOW(), 10, 10, 'NHAN_HON', false, 'Hãy viết gì đó về bản thân...', 1, 1, '', false, $9, true)
			RETURNING id, username, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, ` + db.GuInventoryAggregate("players.id") + `, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, last_mana_update, created_at, lifespan, time_speed_multiplier, party_id, arena_rank, email, google_id, temp_loot, explore_count, last_explore_reset, aperture_stress, last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine, bio, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, is_admin, ref_code, needs_onboarding
		`

		err = tx.QueryRow(queryInsert, username, aptitudeType, aptitudePercent, aptitudeGrade, currentMana, maxMana, googleUser.Email, googleUser.ID, refCode, equippedGus).Scan(
			&player.ID, &player.Username, &player.Rank, &player.SubRank,
			&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
			&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones, &player.GuInventory,
			&player.WallProgress, &player.HumanQi, &player.NextTribulationTime, &player.TribulationCount,
			&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
			&player.EquippedGus, &player.ClearedCells,
			&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries, &player.ActiveCombat,
			&player.LastManaUpdate, &player.CreatedAt,
			&player.Lifespan, &player.TimeSpeedMultiplier, &player.PartyID, &player.ArenaRank,
			&player.Email, &player.GoogleID, &player.TempLoot, &player.ExploreCount, &player.LastExploreReset,
			&player.ApertureStress, &player.LastCultivateTime,
			&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine, &player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.RefCode, &player.NeedsOnboarding,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new player record: " + err.Error()})
			return
		}

		// Persist starter Gus in the player_gu table (the gu_inventory column is gone).
		if err := db.SavePlayerGus(tx, player.ID.String(), starterGu); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to seed starter Gus: " + err.Error()})
			return
		}
		player.GuInventory = starterGu

		// Seed starter materials
		InventorySvc.AddItem(tx, player.ID.String(), "moon_orchid_petal", "materials", 5)
		InventorySvc.AddItem(tx, player.ID.String(), "green_bamboo_wine", "materials", 2)
		InventorySvc.AddItem(tx, player.ID.String(), "jade_stone", "materials", 1)
		// Seed starter recipes
		RecipeSvc.Upsert(tx, &services.RecipeProgress{PlayerID: player.ID.String(), RecipeID: "recipe_moon_raiment", CompletionRate: 100, IsUnlocked: true})
		RecipeSvc.Upsert(tx, &services.RecipeProgress{PlayerID: player.ID.String(), RecipeID: "recipe_white_jade", CompletionRate: 50, IsUnlocked: true})
		RecipeSvc.Upsert(tx, &services.RecipeProgress{PlayerID: player.ID.String(), RecipeID: "recipe_four_flavors_liquor", CompletionRate: 30, IsUnlocked: true})

		// Handle referral code logic
		if referralCode != "" {
			var referrerID uuid.UUID
			err = tx.QueryRow("SELECT id FROM players WHERE ref_code = $1", referralCode).Scan(&referrerID)
			if err == nil {
				if referrerID != player.ID {
					clientIP := c.ClientIP()
					userAgent := c.Request.UserAgent()
					var count int
					_ = tx.QueryRow("SELECT COUNT(*) FROM referrals WHERE referee_ip = $1 AND referee_device = $2", clientIP, userAgent).Scan(&count)
					isSuspicious := count > 0

					_, err = tx.Exec(`
						INSERT INTO referrals (referrer_id, referee_id, referee_ip, referee_device, is_suspicious)
						VALUES ($1, $2, $3, $4, $5)
					`, referrerID, player.ID, clientIP, userAgent, isSuspicious)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record referral: " + err.Error()})
						return
					}
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save Google login state"})
		return
	}

	// Generate JWT session token
	token, err := auth.GenerateToken(player.ID, player.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sign access token"})
		return
	}
	setAuthCookie(c, token)

	// Redirect to frontend
	frontendURL := os.Getenv("CORS_ALLOWED_ORIGINS")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	// Parse the first origin if multiple are defined
	if strings.Contains(frontendURL, ",") {
		frontendURL = strings.Split(frontendURL, ",")[0]
	}

	redirectURL := frontendURL
	if strings.Contains(redirectURL, "?") {
		redirectURL += "&token=" + token
	} else {
		redirectURL += "?token=" + token
	}

	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// TickRegen checks Redis for active regeneration buffs and heals the player
func TickRegen(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	redisKey := fmt.Sprintf("regen_buff:%s", authID.String())
	var itemID string
	err := cache.GetJSON(redisKey, &itemID)
	if err != nil {
		// Key not found or expired
		c.JSON(http.StatusOK, gin.H{"regen_active": false})
		return
	}

	itemSpec, ok := models.ItemRegistry[itemID]
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid item in regen buff"})
		return
	}

	healVal := itemSpec.Effects["regen_hp_out_of_combat"]
	// Divide by 10 since it's meant for a 10-minute duration with 1 tick per minute
	tickHeal := healVal / 10.0
	if tickHeal < 1.0 {
		tickHeal = 1.0
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	if player.HP >= player.BaseHP {
		c.JSON(http.StatusOK, gin.H{"regen_active": true, "message": "HP đã đầy."})
		return
	}

	player.HP += tickHeal
	if player.HP > player.BaseHP {
		player.HP = player.BaseHP
	}

	_, err = tx.Exec("UPDATE players SET hp = $1 WHERE id = $2", player.HP, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save HP"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{
		"regen_active": true,
		"healed":       tickHeal,
		"current_hp":   player.HP,
		"max_hp":       player.BaseHP,
	})
}

func aggregateChestSpecialMessages(specialMessages []string) []string {
	var stones int
	materials := make(map[string]int)
	var others []string
	var chestNames []string

	for _, msg := range specialMessages {
		lines := strings.Split(msg, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// Skip the "Mở <Item> thành công!" or "Mở <Item> thất bại" line as it's redundant
			if strings.HasPrefix(line, "Mở ") && strings.HasSuffix(line, "thành công!") {
				chestName := strings.TrimSuffix(strings.TrimPrefix(line, "Mở "), " thành công!")
				chestNames = append(chestNames, chestName)
				continue
			}

			// 1. Stones pattern: "- 💎 Nhận được %d Nguyên Thạch"
			var amt int
			if _, err := fmt.Sscanf(line, "- 💎 Nhận được %d Nguyên Thạch", &amt); err == nil {
				stones += amt
				continue
			}

			// 2. Material/item pattern: "- Nhận được %dx %s"
			if strings.HasPrefix(line, "- Nhận được ") {
				rest := strings.TrimPrefix(line, "- Nhận được ")
				if idx := strings.Index(rest, "x "); idx != -1 {
					qtyStr := rest[:idx]
					name := rest[idx+2:]
					var q int
					if _, err := fmt.Sscanf(qtyStr, "%d", &q); err == nil {
						materials[name] += q
						continue
					}
				}
			}

			// If not matched, keep it as is
			others = append(others, line)
		}
	}

	var result []string
	if len(chestNames) > 0 {
		counts := make(map[string]int)
		for _, name := range chestNames {
			counts[name]++
		}
		var headers []string
		for name, count := range counts {
			headers = append(headers, fmt.Sprintf("Mở %d lần %s thành công!", count, name))
		}
		result = append(result, headers...)
	}

	if stones > 0 {
		result = append(result, fmt.Sprintf("- 💎 Nhận được %d Nguyên Thạch", stones))
	}

	for name, qty := range materials {
		result = append(result, fmt.Sprintf("- Nhận được %dx %s", qty, name))
	}

	result = append(result, others...)

	return result
}

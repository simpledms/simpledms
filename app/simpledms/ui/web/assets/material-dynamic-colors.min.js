/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */function q(e){return e<0?-1:e===0?0:1}function nt(e,t,r){return(1-r)*e+r*t}function qt(e,t,r){return r<e?e:r>t?t:r}function ht(e,t,r){return r<e?e:r>t?t:r}function Ot(e){return e=e%360,e<0&&(e=e+360),e}function Dt(e){return e=e%360,e<0&&(e=e+360),e}function jt(e,t){return Dt(t-e)<=180?1:-1}function Nt(e,t){return 180-Math.abs(Math.abs(e-t)-180)}function bt(e,t){const r=e[0]*t[0][0]+e[1]*t[0][1]+e[2]*t[0][2],n=e[0]*t[1][0]+e[1]*t[1][1]+e[2]*t[1][2],a=e[0]*t[2][0]+e[1]*t[2][1]+e[2]*t[2][2];return[r,n,a]}/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */const zt=[[.41233895,.35762064,.18051042],[.2126,.7152,.0722],[.01932141,.11916382,.95034478]],Yt=[[3.2413774792388685,-1.5376652402851851,-.49885366846268053],[-.9691452513005321,1.8758853451067872,.04156585616912061],[.05562093689691305,-.20395524564742123,1.0571799111220335]],wt=[95.047,100,108.883];function ft(e,t,r){return(255<<24|(e&255)<<16|(t&255)<<8|r&255)>>>0}function Et(e){const t=tt(e[0]),r=tt(e[1]),n=tt(e[2]);return ft(t,r,n)}function Wt(e){return e>>24&255}function dt(e){return e>>16&255}function mt(e){return e>>8&255}function gt(e){return e&255}function St(e,t,r){const n=Yt,a=n[0][0]*e+n[0][1]*t+n[0][2]*r,o=n[1][0]*e+n[1][1]*t+n[1][2]*r,s=n[2][0]*e+n[2][1]*t+n[2][2]*r,c=tt(a),u=tt(o),h=tt(s);return ft(c,u,h)}function vt(e){const t=$(dt(e)),r=$(mt(e)),n=$(gt(e));return bt([t,r,n],zt)}function Jt(e,t,r){const n=wt,a=(e+16)/116,o=t/500+a,s=a-r/200,c=ut(o),u=ut(a),h=ut(s),l=c*n[0],d=u*n[1],p=h*n[2];return St(l,d,p)}function Xt(e){const t=$(dt(e)),r=$(mt(e)),n=$(gt(e)),a=zt,o=a[0][0]*t+a[0][1]*r+a[0][2]*n,s=a[1][0]*t+a[1][1]*r+a[1][2]*n,c=a[2][0]*t+a[2][1]*r+a[2][2]*n,u=wt,h=o/u[0],l=s/u[1],d=c/u[2],p=at(h),f=at(l),M=at(d),g=116*f-16,b=500*(p-f),w=200*(f-M);return[g,b,w]}function $t(e){const t=Q(e),r=tt(t);return ft(r,r,r)}function Ct(e){const t=vt(e)[1];return 116*at(t/100)-16}function Q(e){return 100*ut((e+16)/116)}function It(e){return at(e/100)*116-16}function $(e){const t=e/255;return t<=.040449936?t/12.92*100:Math.pow((t+.055)/1.055,2.4)*100}function tt(e){const t=e/100;let r=0;return t<=.0031308?r=t*12.92:r=1.055*Math.pow(t,1/2.4)-.055,qt(0,255,Math.round(r*255))}function Kt(){return wt}function at(e){const t=.008856451679035631,r=24389/27;return e>t?Math.pow(e,1/3):(r*e+16)/116}function ut(e){const t=.008856451679035631,r=24389/27,n=e*e*e;return n>t?n:(116*e-16)/r}/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */class v{static make(t=Kt(),r=200/Math.PI*Q(50)/100,n=50,a=2,o=!1){const s=t,c=s[0]*.401288+s[1]*.650173+s[2]*-.051461,u=s[0]*-.250268+s[1]*1.204414+s[2]*.045854,h=s[0]*-.002079+s[1]*.048952+s[2]*.953127,l=.8+a/10,d=l>=.9?nt(.59,.69,(l-.9)*10):nt(.525,.59,(l-.8)*10);let p=o?1:l*(1-1/3.6*Math.exp((-r-42)/92));p=p>1?1:p<0?0:p;const f=l,M=[p*(100/c)+1-p,p*(100/u)+1-p,p*(100/h)+1-p],g=1/(5*r+1),b=g*g*g*g,w=1-b,y=b*r+.1*w*w*Math.cbrt(5*r),k=Q(n)/t[1],A=1.48+Math.sqrt(k),D=.725/Math.pow(k,.2),R=D,P=[Math.pow(y*M[0]*c/100,.42),Math.pow(y*M[1]*u/100,.42),Math.pow(y*M[2]*h/100,.42)],I=[400*P[0]/(P[0]+27.13),400*P[1]/(P[1]+27.13),400*P[2]/(P[2]+27.13)],B=(2*I[0]+I[1]+.05*I[2])*D;return new v(k,B,D,R,d,f,M,y,Math.pow(y,.25),A)}constructor(t,r,n,a,o,s,c,u,h,l){this.n=t,this.aw=r,this.nbb=n,this.ncb=a,this.c=o,this.nc=s,this.rgbD=c,this.fl=u,this.fLRoot=h,this.z=l}}v.DEFAULT=v.make();/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */class z{constructor(t,r,n,a,o,s,c,u,h){this.hue=t,this.chroma=r,this.j=n,this.q=a,this.m=o,this.s=s,this.jstar=c,this.astar=u,this.bstar=h}distance(t){const r=this.jstar-t.jstar,n=this.astar-t.astar,a=this.bstar-t.bstar,o=Math.sqrt(r*r+n*n+a*a);return 1.41*Math.pow(o,.63)}static fromInt(t){return z.fromIntInViewingConditions(t,v.DEFAULT)}static fromIntInViewingConditions(t,r){const n=(t&16711680)>>16,a=(t&65280)>>8,o=t&255,s=$(n),c=$(a),u=$(o),h=.41233895*s+.35762064*c+.18051042*u,l=.2126*s+.7152*c+.0722*u,d=.01932141*s+.11916382*c+.95034478*u,p=.401288*h+.650173*l-.051461*d,f=-.250268*h+1.204414*l+.045854*d,M=-.002079*h+.048952*l+.953127*d,g=r.rgbD[0]*p,b=r.rgbD[1]*f,w=r.rgbD[2]*M,y=Math.pow(r.fl*Math.abs(g)/100,.42),k=Math.pow(r.fl*Math.abs(b)/100,.42),A=Math.pow(r.fl*Math.abs(w)/100,.42),D=q(g)*400*y/(y+27.13),R=q(b)*400*k/(k+27.13),P=q(w)*400*A/(A+27.13),I=(11*D+-12*R+P)/11,B=(D+R-2*P)/9,T=(20*D+20*R+21*P)/20,V=(40*D+20*R+P)/20,U=Math.atan2(B,I)*180/Math.PI,L=U<0?U+360:U>=360?U-360:U,Z=L*Math.PI/180,st=V*r.nbb,K=100*Math.pow(st/r.aw,r.c*r.z),it=4/r.c*Math.sqrt(K/100)*(r.aw+4)*r.fLRoot,pt=L<20.14?L+360:L,yt=.25*(Math.cos(pt*Math.PI/180+2)+3.8),kt=5e4/13*yt*r.nc*r.ncb*Math.sqrt(I*I+B*B)/(T+.305),ct=Math.pow(kt,.9)*Math.pow(1.64-Math.pow(.29,r.n),.73),Bt=ct*Math.sqrt(K/100),Rt=Bt*r.fLRoot,Vt=50*Math.sqrt(ct*r.c/(r.aw+4)),_t=(1+100*.007)*K/(1+.007*K),Ft=1/.0228*Math.log(1+.0228*Rt),Ht=Ft*Math.cos(Z),Ut=Ft*Math.sin(Z);return new z(L,Bt,K,it,Rt,Vt,_t,Ht,Ut)}static fromJch(t,r,n){return z.fromJchInViewingConditions(t,r,n,v.DEFAULT)}static fromJchInViewingConditions(t,r,n,a){const o=4/a.c*Math.sqrt(t/100)*(a.aw+4)*a.fLRoot,s=r*a.fLRoot,c=r/Math.sqrt(t/100),u=50*Math.sqrt(c*a.c/(a.aw+4)),h=n*Math.PI/180,l=(1+100*.007)*t/(1+.007*t),d=1/.0228*Math.log(1+.0228*s),p=d*Math.cos(h),f=d*Math.sin(h);return new z(n,r,t,o,s,u,l,p,f)}static fromUcs(t,r,n){return z.fromUcsInViewingConditions(t,r,n,v.DEFAULT)}static fromUcsInViewingConditions(t,r,n,a){const o=r,s=n,c=Math.sqrt(o*o+s*s),h=(Math.exp(c*.0228)-1)/.0228/a.fLRoot;let l=Math.atan2(s,o)*(180/Math.PI);l<0&&(l+=360);const d=t/(1-(t-100)*.007);return z.fromJchInViewingConditions(d,h,l,a)}toInt(){return this.viewed(v.DEFAULT)}viewed(t){const r=this.chroma===0||this.j===0?0:this.chroma/Math.sqrt(this.j/100),n=Math.pow(r/Math.pow(1.64-Math.pow(.29,t.n),.73),1/.9),a=this.hue*Math.PI/180,o=.25*(Math.cos(a+2)+3.8),s=t.aw*Math.pow(this.j/100,1/t.c/t.z),c=o*(5e4/13)*t.nc*t.ncb,u=s/t.nbb,h=Math.sin(a),l=Math.cos(a),d=23*(u+.305)*n/(23*c+11*n*l+108*n*h),p=d*l,f=d*h,M=(460*u+451*p+288*f)/1403,g=(460*u-891*p-261*f)/1403,b=(460*u-220*p-6300*f)/1403,w=Math.max(0,27.13*Math.abs(M)/(400-Math.abs(M))),y=q(M)*(100/t.fl)*Math.pow(w,1/.42),k=Math.max(0,27.13*Math.abs(g)/(400-Math.abs(g))),A=q(g)*(100/t.fl)*Math.pow(k,1/.42),D=Math.max(0,27.13*Math.abs(b)/(400-Math.abs(b))),R=q(b)*(100/t.fl)*Math.pow(D,1/.42),P=y/t.rgbD[0],I=A/t.rgbD[1],B=R/t.rgbD[2],T=1.86206786*P-1.01125463*I+.14918677*B,V=.38752654*P+.62144744*I-.00897398*B,Y=-.0158415*P-.03412294*I+1.04996444*B;return St(T,V,Y)}static fromXyzInViewingConditions(t,r,n,a){const o=.401288*t+.650173*r-.051461*n,s=-.250268*t+1.204414*r+.045854*n,c=-.002079*t+.048952*r+.953127*n,u=a.rgbD[0]*o,h=a.rgbD[1]*s,l=a.rgbD[2]*c,d=Math.pow(a.fl*Math.abs(u)/100,.42),p=Math.pow(a.fl*Math.abs(h)/100,.42),f=Math.pow(a.fl*Math.abs(l)/100,.42),M=q(u)*400*d/(d+27.13),g=q(h)*400*p/(p+27.13),b=q(l)*400*f/(f+27.13),w=(11*M+-12*g+b)/11,y=(M+g-2*b)/9,k=(20*M+20*g+21*b)/20,A=(40*M+20*g+b)/20,R=Math.atan2(y,w)*180/Math.PI,P=R<0?R+360:R>=360?R-360:R,I=P*Math.PI/180,B=A*a.nbb,T=100*Math.pow(B/a.aw,a.c*a.z),V=4/a.c*Math.sqrt(T/100)*(a.aw+4)*a.fLRoot,Y=P<20.14?P+360:P,U=1/4*(Math.cos(Y*Math.PI/180+2)+3.8),Z=5e4/13*U*a.nc*a.ncb*Math.sqrt(w*w+y*y)/(k+.305),st=Math.pow(Z,.9)*Math.pow(1.64-Math.pow(.29,a.n),.73),K=st*Math.sqrt(T/100),it=K*a.fLRoot,pt=50*Math.sqrt(st*a.c/(a.aw+4)),yt=(1+100*.007)*T/(1+.007*T),Mt=Math.log(1+.0228*it)/.0228,kt=Mt*Math.cos(I),ct=Mt*Math.sin(I);return new z(P,K,T,V,it,pt,yt,kt,ct)}xyzInViewingConditions(t){const r=this.chroma===0||this.j===0?0:this.chroma/Math.sqrt(this.j/100),n=Math.pow(r/Math.pow(1.64-Math.pow(.29,t.n),.73),1/.9),a=this.hue*Math.PI/180,o=.25*(Math.cos(a+2)+3.8),s=t.aw*Math.pow(this.j/100,1/t.c/t.z),c=o*(5e4/13)*t.nc*t.ncb,u=s/t.nbb,h=Math.sin(a),l=Math.cos(a),d=23*(u+.305)*n/(23*c+11*n*l+108*n*h),p=d*l,f=d*h,M=(460*u+451*p+288*f)/1403,g=(460*u-891*p-261*f)/1403,b=(460*u-220*p-6300*f)/1403,w=Math.max(0,27.13*Math.abs(M)/(400-Math.abs(M))),y=q(M)*(100/t.fl)*Math.pow(w,1/.42),k=Math.max(0,27.13*Math.abs(g)/(400-Math.abs(g))),A=q(g)*(100/t.fl)*Math.pow(k,1/.42),D=Math.max(0,27.13*Math.abs(b)/(400-Math.abs(b))),R=q(b)*(100/t.fl)*Math.pow(D,1/.42),P=y/t.rgbD[0],I=A/t.rgbD[1],B=R/t.rgbD[2],T=1.86206786*P-1.01125463*I+.14918677*B,V=.38752654*P+.62144744*I-.00897398*B,Y=-.0158415*P-.03412294*I+1.04996444*B;return[T,V,Y]}}/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */class C{static sanitizeRadians(t){return(t+Math.PI*8)%(Math.PI*2)}static trueDelinearized(t){const r=t/100;let n=0;return r<=.0031308?n=r*12.92:n=1.055*Math.pow(r,1/2.4)-.055,n*255}static chromaticAdaptation(t){const r=Math.pow(Math.abs(t),.42);return q(t)*400*r/(r+27.13)}static hueOf(t){const r=bt(t,C.SCALED_DISCOUNT_FROM_LINRGB),n=C.chromaticAdaptation(r[0]),a=C.chromaticAdaptation(r[1]),o=C.chromaticAdaptation(r[2]),s=(11*n+-12*a+o)/11,c=(n+a-2*o)/9;return Math.atan2(c,s)}static areInCyclicOrder(t,r,n){const a=C.sanitizeRadians(r-t),o=C.sanitizeRadians(n-t);return a<o}static intercept(t,r,n){return(r-t)/(n-t)}static lerpPoint(t,r,n){return[t[0]+(n[0]-t[0])*r,t[1]+(n[1]-t[1])*r,t[2]+(n[2]-t[2])*r]}static setCoordinate(t,r,n,a){const o=C.intercept(t[a],r,n[a]);return C.lerpPoint(t,o,n)}static isBounded(t){return 0<=t&&t<=100}static nthVertex(t,r){const n=C.Y_FROM_LINRGB[0],a=C.Y_FROM_LINRGB[1],o=C.Y_FROM_LINRGB[2],s=r%4<=1?0:100,c=r%2===0?0:100;if(r<4){const u=s,h=c,l=(t-u*a-h*o)/n;return C.isBounded(l)?[l,u,h]:[-1,-1,-1]}else if(r<8){const u=s,h=c,l=(t-h*n-u*o)/a;return C.isBounded(l)?[h,l,u]:[-1,-1,-1]}else{const u=s,h=c,l=(t-u*n-h*a)/o;return C.isBounded(l)?[u,h,l]:[-1,-1,-1]}}static bisectToSegment(t,r){let n=[-1,-1,-1],a=n,o=0,s=0,c=!1,u=!0;for(let h=0;h<12;h++){const l=C.nthVertex(t,h);if(l[0]<0)continue;const d=C.hueOf(l);if(!c){n=l,a=l,o=d,s=d,c=!0;continue}(u||C.areInCyclicOrder(o,d,s))&&(u=!1,C.areInCyclicOrder(o,r,d)?(a=l,s=d):(n=l,o=d))}return[n,a]}static midpoint(t,r){return[(t[0]+r[0])/2,(t[1]+r[1])/2,(t[2]+r[2])/2]}static criticalPlaneBelow(t){return Math.floor(t-.5)}static criticalPlaneAbove(t){return Math.ceil(t-.5)}static bisectToLimit(t,r){const n=C.bisectToSegment(t,r);let a=n[0],o=C.hueOf(a),s=n[1];for(let c=0;c<3;c++)if(a[c]!==s[c]){let u=-1,h=255;a[c]<s[c]?(u=C.criticalPlaneBelow(C.trueDelinearized(a[c])),h=C.criticalPlaneAbove(C.trueDelinearized(s[c]))):(u=C.criticalPlaneAbove(C.trueDelinearized(a[c])),h=C.criticalPlaneBelow(C.trueDelinearized(s[c])));for(let l=0;l<8&&!(Math.abs(h-u)<=1);l++){const d=Math.floor((u+h)/2),p=C.CRITICAL_PLANES[d],f=C.setCoordinate(a,p,s,c),M=C.hueOf(f);C.areInCyclicOrder(o,r,M)?(s=f,h=d):(a=f,o=M,u=d)}}return C.midpoint(a,s)}static inverseChromaticAdaptation(t){const r=Math.abs(t),n=Math.max(0,27.13*r/(400-r));return q(t)*Math.pow(n,1/.42)}static findResultByJ(t,r,n){let a=Math.sqrt(n)*11;const o=v.DEFAULT,s=1/Math.pow(1.64-Math.pow(.29,o.n),.73),u=.25*(Math.cos(t+2)+3.8)*(5e4/13)*o.nc*o.ncb,h=Math.sin(t),l=Math.cos(t);for(let d=0;d<5;d++){const p=a/100,f=r===0||a===0?0:r/Math.sqrt(p),M=Math.pow(f*s,1/.9),b=o.aw*Math.pow(p,1/o.c/o.z)/o.nbb,w=23*(b+.305)*M/(23*u+11*M*l+108*M*h),y=w*l,k=w*h,A=(460*b+451*y+288*k)/1403,D=(460*b-891*y-261*k)/1403,R=(460*b-220*y-6300*k)/1403,P=C.inverseChromaticAdaptation(A),I=C.inverseChromaticAdaptation(D),B=C.inverseChromaticAdaptation(R),T=bt([P,I,B],C.LINRGB_FROM_SCALED_DISCOUNT);if(T[0]<0||T[1]<0||T[2]<0)return 0;const V=C.Y_FROM_LINRGB[0],Y=C.Y_FROM_LINRGB[1],U=C.Y_FROM_LINRGB[2],L=V*T[0]+Y*T[1]+U*T[2];if(L<=0)return 0;if(d===4||Math.abs(L-n)<.002)return T[0]>100.01||T[1]>100.01||T[2]>100.01?0:Et(T);a=a-(L-n)*a/(2*L)}return 0}static solveToInt(t,r,n){if(r<1e-4||n<1e-4||n>99.9999)return $t(n);t=Dt(t);const a=t/180*Math.PI,o=Q(n),s=C.findResultByJ(a,r,o);if(s!==0)return s;const c=C.bisectToLimit(o,a);return Et(c)}static solveToCam(t,r,n){return z.fromInt(C.solveToInt(t,r,n))}}C.SCALED_DISCOUNT_FROM_LINRGB=[[.001200833568784504,.002389694492170889,.0002795742885861124],[.0005891086651375999,.0029785502573438758,.0003270666104008398],[.00010146692491640572,.0005364214359186694,.0032979401770712076]];C.LINRGB_FROM_SCALED_DISCOUNT=[[1373.2198709594231,-1100.4251190754821,-7.278681089101213],[-271.815969077903,559.6580465940733,-32.46047482791194],[1.9622899599665666,-57.173814538844006,308.7233197812385]];C.Y_FROM_LINRGB=[.2126,.7152,.0722];C.CRITICAL_PLANES=[.015176349177441876,.045529047532325624,.07588174588720938,.10623444424209313,.13658714259697685,.16693984095186062,.19729253930674434,.2276452376616281,.2579979360165119,.28835063437139563,.3188300904430532,.350925934958123,.3848314933096426,.42057480301049466,.458183274052838,.4976837250274023,.5391024159806381,.5824650784040898,.6277969426914107,.6751227633498623,.7244668422128921,.775853049866786,.829304845476233,.8848452951698498,.942497089126609,1.0022825574869039,1.0642236851973577,1.1283421258858297,1.1946592148522128,1.2631959812511864,1.3339731595349034,1.407011200216447,1.4823302800086415,1.5599503113873272,1.6398909516233677,1.7221716113234105,1.8068114625156377,1.8938294463134073,1.9832442801866852,2.075074464868551,2.1693382909216234,2.2660538449872063,2.36523901573795,2.4669114995532007,2.5710888059345764,2.6777882626779785,2.7870270208169257,2.898822059350997,3.0131901897720907,3.1301480604002863,3.2497121605402226,3.3718988244681087,3.4967242352587946,3.624204428461639,3.754355295633311,3.887192587735158,4.022731918402185,4.160988767090289,4.301978482107941,4.445716283538092,4.592217266055746,4.741496401646282,4.893568542229298,5.048448422192488,5.20615066083972,5.3666897647573375,5.5300801301023865,5.696336044816294,5.865471690767354,6.037501145825082,6.212438385869475,6.390297286737924,6.571091626112461,6.7548350853498045,6.941541251256611,7.131223617812143,7.323895587840543,7.5195704746346665,7.7182615035334345,7.919981813454504,8.124744458384042,8.332562408825165,8.543448553206703,8.757415699253682,8.974476575321063,9.194643831691977,9.417930041841839,9.644347703669503,9.873909240696694,10.106627003236781,10.342513269534024,10.58158024687427,10.8238400726681,11.069304815507364,11.317986476196008,11.569896988756009,11.825048221409341,12.083451977536606,12.345119996613247,12.610063955123938,12.878295467455942,13.149826086772048,13.42466730586372,13.702830557985108,13.984327217668513,14.269168601521828,14.55736596900856,14.848930523210871,15.143873411576273,15.44220572664832,15.743938506781891,16.04908273684337,16.35764934889634,16.66964922287304,16.985093187232053,17.30399201960269,17.62635644741625,17.95219714852476,18.281524751807332,18.614349837764564,18.95068293910138,19.290534541298456,19.633915083172692,19.98083495742689,20.331304511189067,20.685334046541502,21.042933821039977,21.404114048223256,21.76888489811322,22.137256497705877,22.50923893145328,22.884842241736916,23.264076429332462,23.6469514538663,24.033477234264016,24.42366364919083,24.817520537484558,25.21505769858089,25.61628489293138,26.021211842414342,26.429848230738664,26.842203703840827,27.258287870275353,27.678110301598522,28.10168053274597,28.529008062403893,28.96010235337422,29.39497283293396,29.83362889318845,30.276079891419332,30.722335150426627,31.172403958865512,31.62629557157785,32.08401920991837,32.54558406207592,33.010999283389665,33.4802739966603,33.953417292456834,34.430438229418264,34.911345834551085,35.39614910352207,35.88485700094671,36.37747846067349,36.87402238606382,37.37449765026789,37.87891309649659,38.38727753828926,38.89959975977785,39.41588851594697,39.93615253289054,40.460400508064545,40.98864111053629,41.520882981230194,42.05713473317016,42.597404951718396,43.141702194811224,43.6900349931913,44.24241185063697,44.798841244188324,45.35933162437017,45.92389141541209,46.49252901546552,47.065252796817916,47.64207110610409,48.22299226451468,48.808024568002054,49.3971762874833,49.9904556690408,50.587870934119984,51.189430279724725,51.79514187861014,52.40501387947288,53.0190544071392,53.637271562750364,54.259673423945976,54.88626804504493,55.517063457223934,56.15206766869424,56.79128866487574,57.43473440856916,58.08241284012621,58.734331877617365,59.39049941699807,60.05092333227251,60.715611475655585,61.38457167773311,62.057811747619894,62.7353394731159,63.417162620860914,64.10328893648692,64.79372614476921,65.48848194977529,66.18756403501224,66.89098006357258,67.59873767827808,68.31084450182222,69.02730813691093,69.74813616640164,70.47333615344107,71.20291564160104,71.93688215501312,72.67524319850172,73.41800625771542,74.16517879925733,74.9167682708136,75.67278210128072,76.43322770089146,77.1981124613393,77.96744375590167,78.74122893956174,79.51947534912904,80.30219030335869,81.08938110306934,81.88105503125999,82.67721935322541,83.4778813166706,84.28304815182372,85.09272707154808,85.90692527145302,86.72564993000343,87.54890820862819,88.3767072518277,89.2090541872801,90.04595612594655,90.88742016217518,91.73345337380438,92.58406282226491,93.43925555268066,94.29903859396902,95.16341895893969,96.03240364439274,96.9059996312159,97.78421388448044,98.6670533535366,99.55452497210776];/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */class E{static from(t,r,n){return new E(C.solveToInt(t,r,n))}static fromInt(t){return new E(t)}toInt(){return this.argb}get hue(){return this.internalHue}set hue(t){this.setInternalState(C.solveToInt(t,this.internalChroma,this.internalTone))}get chroma(){return this.internalChroma}set chroma(t){this.setInternalState(C.solveToInt(this.internalHue,t,this.internalTone))}get tone(){return this.internalTone}set tone(t){this.setInternalState(C.solveToInt(this.internalHue,this.internalChroma,t))}constructor(t){this.argb=t;const r=z.fromInt(t);this.internalHue=r.hue,this.internalChroma=r.chroma,this.internalTone=Ct(t),this.argb=t}setInternalState(t){const r=z.fromInt(t);this.internalHue=r.hue,this.internalChroma=r.chroma,this.internalTone=Ct(t),this.argb=t}inViewingConditions(t){const n=z.fromInt(this.toInt()).xyzInViewingConditions(t),a=z.fromXyzInViewingConditions(n[0],n[1],n[2],v.make());return E.from(a.hue,a.chroma,It(n[1]))}}/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */class xt{static harmonize(t,r){const n=E.fromInt(t),a=E.fromInt(r),o=Nt(n.hue,a.hue),s=Math.min(o*.5,15),c=Dt(n.hue+s*jt(n.hue,a.hue));return E.from(c,n.chroma,n.tone).toInt()}static hctHue(t,r,n){const a=xt.cam16Ucs(t,r,n),o=z.fromInt(a),s=z.fromInt(t);return E.from(o.hue,s.chroma,Ct(t)).toInt()}static cam16Ucs(t,r,n){const a=z.fromInt(t),o=z.fromInt(r),s=a.jstar,c=a.astar,u=a.bstar,h=o.jstar,l=o.astar,d=o.bstar,p=s+(h-s)*n,f=c+(l-c)*n,M=u+(d-u)*n;return z.fromUcs(p,f,M).toInt()}}/**
 * @license
 * Copyright 2022 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */class N{static ratioOfTones(t,r){return t=ht(0,100,t),r=ht(0,100,r),N.ratioOfYs(Q(t),Q(r))}static ratioOfYs(t,r){const n=t>r?t:r,a=n===r?t:r;return(n+5)/(a+5)}static lighter(t,r){if(t<0||t>100)return-1;const n=Q(t),a=r*(n+5)-5,o=N.ratioOfYs(a,n),s=Math.abs(o-r);if(o<r&&s>.04)return-1;const c=It(a)+.4;return c<0||c>100?-1:c}static darker(t,r){if(t<0||t>100)return-1;const n=Q(t),a=(n+5)/r-5,o=N.ratioOfYs(n,a),s=Math.abs(o-r);if(o<r&&s>.04)return-1;const c=It(a)-.4;return c<0||c>100?-1:c}static lighterUnsafe(t,r){const n=N.lighter(t,r);return n<0?100:n}static darkerUnsafe(t,r){const n=N.darker(t,r);return n<0?0:n}}/**
 * @license
 * Copyright 2023 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */class At{static isDisliked(t){const r=Math.round(t.hue)>=90&&Math.round(t.hue)<=111,n=Math.round(t.chroma)>16,a=Math.round(t.tone)<65;return r&&n&&a}static fixIfDisliked(t){return At.isDisliked(t)?E.from(t.hue,t.chroma,70):t}}/**
 * @license
 * Copyright 2022 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */class m{static fromPalette(t){return new m(t.name??"",t.palette,t.tone,t.isBackground??!1,t.background,t.secondBackground,t.contrastCurve,t.toneDeltaPair)}constructor(t,r,n,a,o,s,c,u){if(this.name=t,this.palette=r,this.tone=n,this.isBackground=a,this.background=o,this.secondBackground=s,this.contrastCurve=c,this.toneDeltaPair=u,this.hctCache=new Map,!o&&s)throw new Error(`Color ${t} has secondBackgrounddefined, but background is not defined.`);if(!o&&c)throw new Error(`Color ${t} has contrastCurvedefined, but background is not defined.`);if(o&&!c)throw new Error(`Color ${t} has backgrounddefined, but contrastCurve is not defined.`)}getArgb(t){return this.getHct(t).toInt()}getHct(t){const r=this.hctCache.get(t);if(r!=null)return r;const n=this.getTone(t),a=this.palette(t).getHct(n);return this.hctCache.size>4&&this.hctCache.clear(),this.hctCache.set(t,a),a}getTone(t){const r=t.contrastLevel<0;if(this.toneDeltaPair){const n=this.toneDeltaPair(t),a=n.roleA,o=n.roleB,s=n.delta,c=n.polarity,u=n.stayTogether,l=this.background(t).getTone(t),d=c==="nearer"||c==="lighter"&&!t.isDark||c==="darker"&&t.isDark,p=d?a:o,f=d?o:a,M=this.name===p.name,g=t.isDark?1:-1,b=p.contrastCurve.getContrast(t.contrastLevel),w=f.contrastCurve.getContrast(t.contrastLevel),y=p.tone(t);let k=N.ratioOfTones(l,y)>=b?y:m.foregroundTone(l,b);const A=f.tone(t);let D=N.ratioOfTones(l,A)>=w?A:m.foregroundTone(l,w);return r&&(k=m.foregroundTone(l,b),D=m.foregroundTone(l,w)),(D-k)*g>=s||(D=ht(0,100,k+s*g),(D-k)*g>=s||(k=ht(0,100,D-s*g))),50<=k&&k<60?g>0?(k=60,D=Math.max(D,k+s*g)):(k=49,D=Math.min(D,k+s*g)):50<=D&&D<60&&(u?g>0?(k=60,D=Math.max(D,k+s*g)):(k=49,D=Math.min(D,k+s*g)):g>0?D=60:D=49),M?k:D}else{let n=this.tone(t);if(this.background==null)return n;const a=this.background(t).getTone(t),o=this.contrastCurve.getContrast(t.contrastLevel);if(N.ratioOfTones(a,n)>=o||(n=m.foregroundTone(a,o)),r&&(n=m.foregroundTone(a,o)),this.isBackground&&50<=n&&n<60&&(N.ratioOfTones(49,a)>=o?n=49:n=60),this.secondBackground){const[s,c]=[this.background,this.secondBackground],[u,h]=[s(t).getTone(t),c(t).getTone(t)],[l,d]=[Math.max(u,h),Math.min(u,h)];if(N.ratioOfTones(l,n)>=o&&N.ratioOfTones(d,n)>=o)return n;const p=N.lighter(l,o),f=N.darker(d,o),M=[];return p!==-1&&M.push(p),f!==-1&&M.push(f),m.tonePrefersLightForeground(u)||m.tonePrefersLightForeground(h)?p<0?100:p:M.length===1?M[0]:f<0?0:f}return n}}static foregroundTone(t,r){const n=N.lighterUnsafe(t,r),a=N.darkerUnsafe(t,r),o=N.ratioOfTones(n,t),s=N.ratioOfTones(a,t);if(m.tonePrefersLightForeground(t)){const u=Math.abs(o-s)<.1&&o<r&&s<r;return o>=r||o>=s||u?n:a}else return s>=r||s>=o?a:n}static tonePrefersLightForeground(t){return Math.round(t)<60}static toneAllowsLightForeground(t){return Math.round(t)<=49}static enableLightForeground(t){return m.tonePrefersLightForeground(t)&&!m.toneAllowsLightForeground(t)?49:t}}/**
 * @license
 * Copyright 2022 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */var ot;(function(e){e[e.MONOCHROME=0]="MONOCHROME",e[e.NEUTRAL=1]="NEUTRAL",e[e.TONAL_SPOT=2]="TONAL_SPOT",e[e.VIBRANT=3]="VIBRANT",e[e.EXPRESSIVE=4]="EXPRESSIVE",e[e.FIDELITY=5]="FIDELITY",e[e.CONTENT=6]="CONTENT",e[e.RAINBOW=7]="RAINBOW",e[e.FRUIT_SALAD=8]="FRUIT_SALAD"})(ot||(ot={}));/**
 * @license
 * Copyright 2023 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */class x{constructor(t,r,n,a){this.low=t,this.normal=r,this.medium=n,this.high=a}getContrast(t){return t<=-1?this.low:t<0?nt(this.low,this.normal,(t- -1)/1):t<.5?nt(this.normal,this.medium,(t-0)/.5):t<1?nt(this.medium,this.high,(t-.5)/.5):this.high}}/**
 * @license
 * Copyright 2023 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */class j{constructor(t,r,n,a,o){this.roleA=t,this.roleB=r,this.delta=n,this.polarity=a,this.stayTogether=o}}/**
 * @license
 * Copyright 2022 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */function et(e){return e.variant===ot.FIDELITY||e.variant===ot.CONTENT}function O(e){return e.variant===ot.MONOCHROME}function Qt(e,t,r,n){let a=r,o=E.from(e,t,r);if(o.chroma<t){let s=o.chroma;for(;o.chroma<t;){a+=n?-1:1;const c=E.from(e,t,a);if(s>c.chroma||Math.abs(c.chroma-t)<.4)break;const u=Math.abs(c.chroma-t),h=Math.abs(o.chroma-t);u<h&&(o=c),s=Math.max(s,c.chroma)}}return a}function Zt(e){return v.make(void 0,void 0,e.isDark?30:80,void 0,void 0)}function Tt(e,t){const r=e.inViewingConditions(Zt(t));return m.tonePrefersLightForeground(e.tone)&&!m.toneAllowsLightForeground(r.tone)?m.enableLightForeground(e.tone):m.enableLightForeground(r.tone)}class i{static highestSurface(t){return t.isDark?i.surfaceBright:i.surfaceDim}}i.contentAccentToneDelta=15;i.primaryPaletteKeyColor=m.fromPalette({name:"primary_palette_key_color",palette:e=>e.primaryPalette,tone:e=>e.primaryPalette.keyColor.tone});i.secondaryPaletteKeyColor=m.fromPalette({name:"secondary_palette_key_color",palette:e=>e.secondaryPalette,tone:e=>e.secondaryPalette.keyColor.tone});i.tertiaryPaletteKeyColor=m.fromPalette({name:"tertiary_palette_key_color",palette:e=>e.tertiaryPalette,tone:e=>e.tertiaryPalette.keyColor.tone});i.neutralPaletteKeyColor=m.fromPalette({name:"neutral_palette_key_color",palette:e=>e.neutralPalette,tone:e=>e.neutralPalette.keyColor.tone});i.neutralVariantPaletteKeyColor=m.fromPalette({name:"neutral_variant_palette_key_color",palette:e=>e.neutralVariantPalette,tone:e=>e.neutralVariantPalette.keyColor.tone});i.background=m.fromPalette({name:"background",palette:e=>e.neutralPalette,tone:e=>e.isDark?6:98,isBackground:!0});i.onBackground=m.fromPalette({name:"on_background",palette:e=>e.neutralPalette,tone:e=>e.isDark?90:10,background:e=>i.background,contrastCurve:new x(3,3,4.5,7)});i.surface=m.fromPalette({name:"surface",palette:e=>e.neutralPalette,tone:e=>e.isDark?6:98,isBackground:!0});i.surfaceDim=m.fromPalette({name:"surface_dim",palette:e=>e.neutralPalette,tone:e=>e.isDark?6:87,isBackground:!0});i.surfaceBright=m.fromPalette({name:"surface_bright",palette:e=>e.neutralPalette,tone:e=>e.isDark?24:98,isBackground:!0});i.surfaceContainerLowest=m.fromPalette({name:"surface_container_lowest",palette:e=>e.neutralPalette,tone:e=>e.isDark?4:100,isBackground:!0});i.surfaceContainerLow=m.fromPalette({name:"surface_container_low",palette:e=>e.neutralPalette,tone:e=>e.isDark?10:96,isBackground:!0});i.surfaceContainer=m.fromPalette({name:"surface_container",palette:e=>e.neutralPalette,tone:e=>e.isDark?12:94,isBackground:!0});i.surfaceContainerHigh=m.fromPalette({name:"surface_container_high",palette:e=>e.neutralPalette,tone:e=>e.isDark?17:92,isBackground:!0});i.surfaceContainerHighest=m.fromPalette({name:"surface_container_highest",palette:e=>e.neutralPalette,tone:e=>e.isDark?22:90,isBackground:!0});i.onSurface=m.fromPalette({name:"on_surface",palette:e=>e.neutralPalette,tone:e=>e.isDark?90:10,background:e=>i.highestSurface(e),contrastCurve:new x(4.5,7,11,21)});i.surfaceVariant=m.fromPalette({name:"surface_variant",palette:e=>e.neutralVariantPalette,tone:e=>e.isDark?30:90,isBackground:!0});i.onSurfaceVariant=m.fromPalette({name:"on_surface_variant",palette:e=>e.neutralVariantPalette,tone:e=>e.isDark?80:30,background:e=>i.highestSurface(e),contrastCurve:new x(3,4.5,7,11)});i.inverseSurface=m.fromPalette({name:"inverse_surface",palette:e=>e.neutralPalette,tone:e=>e.isDark?90:20});i.inverseOnSurface=m.fromPalette({name:"inverse_on_surface",palette:e=>e.neutralPalette,tone:e=>e.isDark?20:95,background:e=>i.inverseSurface,contrastCurve:new x(4.5,7,11,21)});i.outline=m.fromPalette({name:"outline",palette:e=>e.neutralVariantPalette,tone:e=>e.isDark?60:50,background:e=>i.highestSurface(e),contrastCurve:new x(1.5,3,4.5,7)});i.outlineVariant=m.fromPalette({name:"outline_variant",palette:e=>e.neutralVariantPalette,tone:e=>e.isDark?30:80,background:e=>i.highestSurface(e),contrastCurve:new x(1,1,3,7)});i.shadow=m.fromPalette({name:"shadow",palette:e=>e.neutralPalette,tone:e=>0});i.scrim=m.fromPalette({name:"scrim",palette:e=>e.neutralPalette,tone:e=>0});i.surfaceTint=m.fromPalette({name:"surface_tint",palette:e=>e.primaryPalette,tone:e=>e.isDark?80:40,isBackground:!0});i.primary=m.fromPalette({name:"primary",palette:e=>e.primaryPalette,tone:e=>O(e)?e.isDark?100:0:e.isDark?80:40,isBackground:!0,background:e=>i.highestSurface(e),contrastCurve:new x(3,4.5,7,11),toneDeltaPair:e=>new j(i.primaryContainer,i.primary,15,"nearer",!1)});i.onPrimary=m.fromPalette({name:"on_primary",palette:e=>e.primaryPalette,tone:e=>O(e)?e.isDark?10:90:e.isDark?20:100,background:e=>i.primary,contrastCurve:new x(4.5,7,11,21)});i.primaryContainer=m.fromPalette({name:"primary_container",palette:e=>e.primaryPalette,tone:e=>et(e)?Tt(e.sourceColorHct,e):O(e)?e.isDark?85:25:e.isDark?30:90,isBackground:!0,background:e=>i.highestSurface(e),contrastCurve:new x(1,1,3,7),toneDeltaPair:e=>new j(i.primaryContainer,i.primary,15,"nearer",!1)});i.onPrimaryContainer=m.fromPalette({name:"on_primary_container",palette:e=>e.primaryPalette,tone:e=>et(e)?m.foregroundTone(i.primaryContainer.tone(e),4.5):O(e)?e.isDark?0:100:e.isDark?90:10,background:e=>i.primaryContainer,contrastCurve:new x(4.5,7,11,21)});i.inversePrimary=m.fromPalette({name:"inverse_primary",palette:e=>e.primaryPalette,tone:e=>e.isDark?40:80,background:e=>i.inverseSurface,contrastCurve:new x(3,4.5,7,11)});i.secondary=m.fromPalette({name:"secondary",palette:e=>e.secondaryPalette,tone:e=>e.isDark?80:40,isBackground:!0,background:e=>i.highestSurface(e),contrastCurve:new x(3,4.5,7,11),toneDeltaPair:e=>new j(i.secondaryContainer,i.secondary,15,"nearer",!1)});i.onSecondary=m.fromPalette({name:"on_secondary",palette:e=>e.secondaryPalette,tone:e=>O(e)?e.isDark?10:100:e.isDark?20:100,background:e=>i.secondary,contrastCurve:new x(4.5,7,11,21)});i.secondaryContainer=m.fromPalette({name:"secondary_container",palette:e=>e.secondaryPalette,tone:e=>{const t=e.isDark?30:90;if(O(e))return e.isDark?30:85;if(!et(e))return t;let r=Qt(e.secondaryPalette.hue,e.secondaryPalette.chroma,t,!e.isDark);return r=Tt(e.secondaryPalette.getHct(r),e),r},isBackground:!0,background:e=>i.highestSurface(e),contrastCurve:new x(1,1,3,7),toneDeltaPair:e=>new j(i.secondaryContainer,i.secondary,15,"nearer",!1)});i.onSecondaryContainer=m.fromPalette({name:"on_secondary_container",palette:e=>e.secondaryPalette,tone:e=>et(e)?m.foregroundTone(i.secondaryContainer.tone(e),4.5):e.isDark?90:10,background:e=>i.secondaryContainer,contrastCurve:new x(4.5,7,11,21)});i.tertiary=m.fromPalette({name:"tertiary",palette:e=>e.tertiaryPalette,tone:e=>O(e)?e.isDark?90:25:e.isDark?80:40,isBackground:!0,background:e=>i.highestSurface(e),contrastCurve:new x(3,4.5,7,11),toneDeltaPair:e=>new j(i.tertiaryContainer,i.tertiary,15,"nearer",!1)});i.onTertiary=m.fromPalette({name:"on_tertiary",palette:e=>e.tertiaryPalette,tone:e=>O(e)?e.isDark?10:90:e.isDark?20:100,background:e=>i.tertiary,contrastCurve:new x(4.5,7,11,21)});i.tertiaryContainer=m.fromPalette({name:"tertiary_container",palette:e=>e.tertiaryPalette,tone:e=>{if(O(e))return e.isDark?60:49;if(!et(e))return e.isDark?30:90;const t=Tt(e.tertiaryPalette.getHct(e.sourceColorHct.tone),e),r=e.tertiaryPalette.getHct(t);return At.fixIfDisliked(r).tone},isBackground:!0,background:e=>i.highestSurface(e),contrastCurve:new x(1,1,3,7),toneDeltaPair:e=>new j(i.tertiaryContainer,i.tertiary,15,"nearer",!1)});i.onTertiaryContainer=m.fromPalette({name:"on_tertiary_container",palette:e=>e.tertiaryPalette,tone:e=>O(e)?e.isDark?0:100:et(e)?m.foregroundTone(i.tertiaryContainer.tone(e),4.5):e.isDark?90:10,background:e=>i.tertiaryContainer,contrastCurve:new x(4.5,7,11,21)});i.error=m.fromPalette({name:"error",palette:e=>e.errorPalette,tone:e=>e.isDark?80:40,isBackground:!0,background:e=>i.highestSurface(e),contrastCurve:new x(3,4.5,7,11),toneDeltaPair:e=>new j(i.errorContainer,i.error,15,"nearer",!1)});i.onError=m.fromPalette({name:"on_error",palette:e=>e.errorPalette,tone:e=>e.isDark?20:100,background:e=>i.error,contrastCurve:new x(4.5,7,11,21)});i.errorContainer=m.fromPalette({name:"error_container",palette:e=>e.errorPalette,tone:e=>e.isDark?30:90,isBackground:!0,background:e=>i.highestSurface(e),contrastCurve:new x(1,1,3,7),toneDeltaPair:e=>new j(i.errorContainer,i.error,15,"nearer",!1)});i.onErrorContainer=m.fromPalette({name:"on_error_container",palette:e=>e.errorPalette,tone:e=>e.isDark?90:10,background:e=>i.errorContainer,contrastCurve:new x(4.5,7,11,21)});i.primaryFixed=m.fromPalette({name:"primary_fixed",palette:e=>e.primaryPalette,tone:e=>O(e)?40:90,isBackground:!0,background:e=>i.highestSurface(e),contrastCurve:new x(1,1,3,7),toneDeltaPair:e=>new j(i.primaryFixed,i.primaryFixedDim,10,"lighter",!0)});i.primaryFixedDim=m.fromPalette({name:"primary_fixed_dim",palette:e=>e.primaryPalette,tone:e=>O(e)?30:80,isBackground:!0,background:e=>i.highestSurface(e),contrastCurve:new x(1,1,3,7),toneDeltaPair:e=>new j(i.primaryFixed,i.primaryFixedDim,10,"lighter",!0)});i.onPrimaryFixed=m.fromPalette({name:"on_primary_fixed",palette:e=>e.primaryPalette,tone:e=>O(e)?100:10,background:e=>i.primaryFixedDim,secondBackground:e=>i.primaryFixed,contrastCurve:new x(4.5,7,11,21)});i.onPrimaryFixedVariant=m.fromPalette({name:"on_primary_fixed_variant",palette:e=>e.primaryPalette,tone:e=>O(e)?90:30,background:e=>i.primaryFixedDim,secondBackground:e=>i.primaryFixed,contrastCurve:new x(3,4.5,7,11)});i.secondaryFixed=m.fromPalette({name:"secondary_fixed",palette:e=>e.secondaryPalette,tone:e=>O(e)?80:90,isBackground:!0,background:e=>i.highestSurface(e),contrastCurve:new x(1,1,3,7),toneDeltaPair:e=>new j(i.secondaryFixed,i.secondaryFixedDim,10,"lighter",!0)});i.secondaryFixedDim=m.fromPalette({name:"secondary_fixed_dim",palette:e=>e.secondaryPalette,tone:e=>O(e)?70:80,isBackground:!0,background:e=>i.highestSurface(e),contrastCurve:new x(1,1,3,7),toneDeltaPair:e=>new j(i.secondaryFixed,i.secondaryFixedDim,10,"lighter",!0)});i.onSecondaryFixed=m.fromPalette({name:"on_secondary_fixed",palette:e=>e.secondaryPalette,tone:e=>10,background:e=>i.secondaryFixedDim,secondBackground:e=>i.secondaryFixed,contrastCurve:new x(4.5,7,11,21)});i.onSecondaryFixedVariant=m.fromPalette({name:"on_secondary_fixed_variant",palette:e=>e.secondaryPalette,tone:e=>O(e)?25:30,background:e=>i.secondaryFixedDim,secondBackground:e=>i.secondaryFixed,contrastCurve:new x(3,4.5,7,11)});i.tertiaryFixed=m.fromPalette({name:"tertiary_fixed",palette:e=>e.tertiaryPalette,tone:e=>O(e)?40:90,isBackground:!0,background:e=>i.highestSurface(e),contrastCurve:new x(1,1,3,7),toneDeltaPair:e=>new j(i.tertiaryFixed,i.tertiaryFixedDim,10,"lighter",!0)});i.tertiaryFixedDim=m.fromPalette({name:"tertiary_fixed_dim",palette:e=>e.tertiaryPalette,tone:e=>O(e)?30:80,isBackground:!0,background:e=>i.highestSurface(e),contrastCurve:new x(1,1,3,7),toneDeltaPair:e=>new j(i.tertiaryFixed,i.tertiaryFixedDim,10,"lighter",!0)});i.onTertiaryFixed=m.fromPalette({name:"on_tertiary_fixed",palette:e=>e.tertiaryPalette,tone:e=>O(e)?100:10,background:e=>i.tertiaryFixedDim,secondBackground:e=>i.tertiaryFixed,contrastCurve:new x(4.5,7,11,21)});i.onTertiaryFixedVariant=m.fromPalette({name:"on_tertiary_fixed_variant",palette:e=>e.tertiaryPalette,tone:e=>O(e)?90:30,background:e=>i.tertiaryFixedDim,secondBackground:e=>i.tertiaryFixed,contrastCurve:new x(3,4.5,7,11)});/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */class G{static fromInt(t){const r=E.fromInt(t);return G.fromHct(r)}static fromHct(t){return new G(t.hue,t.chroma,t)}static fromHueAndChroma(t,r){return new G(t,r,G.createKeyColor(t,r))}constructor(t,r,n){this.hue=t,this.chroma=r,this.keyColor=n,this.cache=new Map}static createKeyColor(t,r){let a=E.from(t,r,50),o=Math.abs(a.chroma-r);for(let s=1;s<50;s+=1){if(Math.round(r)===Math.round(a.chroma))return a;const c=E.from(t,r,50+s),u=Math.abs(c.chroma-r);u<o&&(o=u,a=c);const h=E.from(t,r,50-s),l=Math.abs(h.chroma-r);l<o&&(o=l,a=h)}return a}tone(t){let r=this.cache.get(t);return r===void 0&&(r=E.from(this.hue,this.chroma,t).toInt(),this.cache.set(t,r)),r}getHct(t){return E.fromInt(this.tone(t))}}/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */class S{static of(t){return new S(t,!1)}static contentOf(t){return new S(t,!0)}static fromColors(t){return S.createPaletteFromColors(!1,t)}static contentFromColors(t){return S.createPaletteFromColors(!0,t)}static createPaletteFromColors(t,r){const n=new S(r.primary,t);if(r.secondary){const a=new S(r.secondary,t);n.a2=a.a1}if(r.tertiary){const a=new S(r.tertiary,t);n.a3=a.a1}if(r.error){const a=new S(r.error,t);n.error=a.a1}if(r.neutral){const a=new S(r.neutral,t);n.n1=a.n1}if(r.neutralVariant){const a=new S(r.neutralVariant,t);n.n2=a.n2}return n}constructor(t,r){const n=E.fromInt(t),a=n.hue,o=n.chroma;r?(this.a1=G.fromHueAndChroma(a,o),this.a2=G.fromHueAndChroma(a,o/3),this.a3=G.fromHueAndChroma(a+60,o/2),this.n1=G.fromHueAndChroma(a,Math.min(o/12,4)),this.n2=G.fromHueAndChroma(a,Math.min(o/6,8))):(this.a1=G.fromHueAndChroma(a,Math.max(48,o)),this.a2=G.fromHueAndChroma(a,16),this.a3=G.fromHueAndChroma(a+60,24),this.n1=G.fromHueAndChroma(a,4),this.n2=G.fromHueAndChroma(a,8)),this.error=G.fromHueAndChroma(25,84)}}/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */class te{fromInt(t){return Xt(t)}toInt(t){return Jt(t[0],t[1],t[2])}distance(t,r){const n=t[0]-r[0],a=t[1]-r[1],o=t[2]-r[2];return n*n+a*a+o*o}}/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */const ee=10,re=3;class ne{static quantize(t,r,n){const a=new Map,o=new Array,s=new Array,c=new te;let u=0;for(let y=0;y<t.length;y++){const k=t[y],A=a.get(k);A===void 0?(u++,o.push(c.fromInt(k)),s.push(k),a.set(k,1)):a.set(k,A+1)}const h=new Array;for(let y=0;y<u;y++){const k=s[y],A=a.get(k);A!==void 0&&(h[y]=A)}let l=Math.min(n,u);r.length>0&&(l=Math.min(l,r.length));const d=new Array;for(let y=0;y<r.length;y++)d.push(c.fromInt(r[y]));const p=l-d.length;if(r.length===0&&p>0)for(let y=0;y<p;y++){const k=Math.random()*100,A=Math.random()*(100- -100+1)+-100,D=Math.random()*(100- -100+1)+-100;d.push(new Array(k,A,D))}const f=new Array;for(let y=0;y<u;y++)f.push(Math.floor(Math.random()*l));const M=new Array;for(let y=0;y<l;y++){M.push(new Array);for(let k=0;k<l;k++)M[y].push(0)}const g=new Array;for(let y=0;y<l;y++){g.push(new Array);for(let k=0;k<l;k++)g[y].push(new ae)}const b=new Array;for(let y=0;y<l;y++)b.push(0);for(let y=0;y<ee;y++){for(let P=0;P<l;P++){for(let I=P+1;I<l;I++){const B=c.distance(d[P],d[I]);g[I][P].distance=B,g[I][P].index=P,g[P][I].distance=B,g[P][I].index=I}g[P].sort();for(let I=0;I<l;I++)M[P][I]=g[P][I].index}let k=0;for(let P=0;P<u;P++){const I=o[P],B=f[P],T=d[B],V=c.distance(I,T);let Y=V,U=-1;for(let L=0;L<l;L++){if(g[B][L].distance>=4*V)continue;const Z=c.distance(I,d[L]);Z<Y&&(Y=Z,U=L)}U!==-1&&Math.abs(Math.sqrt(Y)-Math.sqrt(V))>re&&(k++,f[P]=U)}if(k===0&&y!==0)break;const A=new Array(l).fill(0),D=new Array(l).fill(0),R=new Array(l).fill(0);for(let P=0;P<l;P++)b[P]=0;for(let P=0;P<u;P++){const I=f[P],B=o[P],T=h[P];b[I]+=T,A[I]+=B[0]*T,D[I]+=B[1]*T,R[I]+=B[2]*T}for(let P=0;P<l;P++){const I=b[P];if(I===0){d[P]=[0,0,0];continue}const B=A[P]/I,T=D[P]/I,V=R[P]/I;d[P]=[B,T,V]}}const w=new Map;for(let y=0;y<l;y++){const k=b[y];if(k===0)continue;const A=c.toInt(d[y]);w.has(A)||w.set(A,k)}return w}}class ae{constructor(){this.distance=-1,this.index=-1}}/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */class oe{static quantize(t){const r=new Map;for(let n=0;n<t.length;n++){const a=t[n];Wt(a)<255||r.set(a,(r.get(a)??0)+1)}return r}}/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */const lt=5,W=33,rt=35937,_={RED:"red",GREEN:"green",BLUE:"blue"};class se{constructor(t=[],r=[],n=[],a=[],o=[],s=[]){this.weights=t,this.momentsR=r,this.momentsG=n,this.momentsB=a,this.moments=o,this.cubes=s}quantize(t,r){this.constructHistogram(t),this.computeMoments();const n=this.createBoxes(r);return this.createResult(n.resultCount)}constructHistogram(t){this.weights=Array.from({length:rt}).fill(0),this.momentsR=Array.from({length:rt}).fill(0),this.momentsG=Array.from({length:rt}).fill(0),this.momentsB=Array.from({length:rt}).fill(0),this.moments=Array.from({length:rt}).fill(0);const r=oe.quantize(t);for(const[n,a]of r.entries()){const o=dt(n),s=mt(n),c=gt(n),u=8-lt,h=(o>>u)+1,l=(s>>u)+1,d=(c>>u)+1,p=this.getIndex(h,l,d);this.weights[p]=(this.weights[p]??0)+a,this.momentsR[p]+=a*o,this.momentsG[p]+=a*s,this.momentsB[p]+=a*c,this.moments[p]+=a*(o*o+s*s+c*c)}}computeMoments(){for(let t=1;t<W;t++){const r=Array.from({length:W}).fill(0),n=Array.from({length:W}).fill(0),a=Array.from({length:W}).fill(0),o=Array.from({length:W}).fill(0),s=Array.from({length:W}).fill(0);for(let c=1;c<W;c++){let u=0,h=0,l=0,d=0,p=0;for(let f=1;f<W;f++){const M=this.getIndex(t,c,f);u+=this.weights[M],h+=this.momentsR[M],l+=this.momentsG[M],d+=this.momentsB[M],p+=this.moments[M],r[f]+=u,n[f]+=h,a[f]+=l,o[f]+=d,s[f]+=p;const g=this.getIndex(t-1,c,f);this.weights[M]=this.weights[g]+r[f],this.momentsR[M]=this.momentsR[g]+n[f],this.momentsG[M]=this.momentsG[g]+a[f],this.momentsB[M]=this.momentsB[g]+o[f],this.moments[M]=this.moments[g]+s[f]}}}}createBoxes(t){this.cubes=Array.from({length:t}).fill(0).map(()=>new ie);const r=Array.from({length:t}).fill(0);this.cubes[0].r0=0,this.cubes[0].g0=0,this.cubes[0].b0=0,this.cubes[0].r1=W-1,this.cubes[0].g1=W-1,this.cubes[0].b1=W-1;let n=t,a=0;for(let o=1;o<t;o++){this.cut(this.cubes[a],this.cubes[o])?(r[a]=this.cubes[a].vol>1?this.variance(this.cubes[a]):0,r[o]=this.cubes[o].vol>1?this.variance(this.cubes[o]):0):(r[a]=0,o--),a=0;let s=r[0];for(let c=1;c<=o;c++)r[c]>s&&(s=r[c],a=c);if(s<=0){n=o+1;break}}return new ce(t,n)}createResult(t){const r=[];for(let n=0;n<t;++n){const a=this.cubes[n],o=this.volume(a,this.weights);if(o>0){const s=Math.round(this.volume(a,this.momentsR)/o),c=Math.round(this.volume(a,this.momentsG)/o),u=Math.round(this.volume(a,this.momentsB)/o),h=255<<24|(s&255)<<16|(c&255)<<8|u&255;r.push(h)}}return r}variance(t){const r=this.volume(t,this.momentsR),n=this.volume(t,this.momentsG),a=this.volume(t,this.momentsB),o=this.moments[this.getIndex(t.r1,t.g1,t.b1)]-this.moments[this.getIndex(t.r1,t.g1,t.b0)]-this.moments[this.getIndex(t.r1,t.g0,t.b1)]+this.moments[this.getIndex(t.r1,t.g0,t.b0)]-this.moments[this.getIndex(t.r0,t.g1,t.b1)]+this.moments[this.getIndex(t.r0,t.g1,t.b0)]+this.moments[this.getIndex(t.r0,t.g0,t.b1)]-this.moments[this.getIndex(t.r0,t.g0,t.b0)],s=r*r+n*n+a*a,c=this.volume(t,this.weights);return o-s/c}cut(t,r){const n=this.volume(t,this.momentsR),a=this.volume(t,this.momentsG),o=this.volume(t,this.momentsB),s=this.volume(t,this.weights),c=this.maximize(t,_.RED,t.r0+1,t.r1,n,a,o,s),u=this.maximize(t,_.GREEN,t.g0+1,t.g1,n,a,o,s),h=this.maximize(t,_.BLUE,t.b0+1,t.b1,n,a,o,s);let l;const d=c.maximum,p=u.maximum,f=h.maximum;if(d>=p&&d>=f){if(c.cutLocation<0)return!1;l=_.RED}else p>=d&&p>=f?l=_.GREEN:l=_.BLUE;switch(r.r1=t.r1,r.g1=t.g1,r.b1=t.b1,l){case _.RED:t.r1=c.cutLocation,r.r0=t.r1,r.g0=t.g0,r.b0=t.b0;break;case _.GREEN:t.g1=u.cutLocation,r.r0=t.r0,r.g0=t.g1,r.b0=t.b0;break;case _.BLUE:t.b1=h.cutLocation,r.r0=t.r0,r.g0=t.g0,r.b0=t.b1;break;default:throw new Error("unexpected direction "+l)}return t.vol=(t.r1-t.r0)*(t.g1-t.g0)*(t.b1-t.b0),r.vol=(r.r1-r.r0)*(r.g1-r.g0)*(r.b1-r.b0),!0}maximize(t,r,n,a,o,s,c,u){const h=this.bottom(t,r,this.momentsR),l=this.bottom(t,r,this.momentsG),d=this.bottom(t,r,this.momentsB),p=this.bottom(t,r,this.weights);let f=0,M=-1,g=0,b=0,w=0,y=0;for(let k=n;k<a;k++){if(g=h+this.top(t,r,k,this.momentsR),b=l+this.top(t,r,k,this.momentsG),w=d+this.top(t,r,k,this.momentsB),y=p+this.top(t,r,k,this.weights),y===0)continue;let A=(g*g+b*b+w*w)*1,D=y*1,R=A/D;g=o-g,b=s-b,w=c-w,y=u-y,y!==0&&(A=(g*g+b*b+w*w)*1,D=y*1,R+=A/D,R>f&&(f=R,M=k))}return new le(M,f)}volume(t,r){return r[this.getIndex(t.r1,t.g1,t.b1)]-r[this.getIndex(t.r1,t.g1,t.b0)]-r[this.getIndex(t.r1,t.g0,t.b1)]+r[this.getIndex(t.r1,t.g0,t.b0)]-r[this.getIndex(t.r0,t.g1,t.b1)]+r[this.getIndex(t.r0,t.g1,t.b0)]+r[this.getIndex(t.r0,t.g0,t.b1)]-r[this.getIndex(t.r0,t.g0,t.b0)]}bottom(t,r,n){switch(r){case _.RED:return-n[this.getIndex(t.r0,t.g1,t.b1)]+n[this.getIndex(t.r0,t.g1,t.b0)]+n[this.getIndex(t.r0,t.g0,t.b1)]-n[this.getIndex(t.r0,t.g0,t.b0)];case _.GREEN:return-n[this.getIndex(t.r1,t.g0,t.b1)]+n[this.getIndex(t.r1,t.g0,t.b0)]+n[this.getIndex(t.r0,t.g0,t.b1)]-n[this.getIndex(t.r0,t.g0,t.b0)];case _.BLUE:return-n[this.getIndex(t.r1,t.g1,t.b0)]+n[this.getIndex(t.r1,t.g0,t.b0)]+n[this.getIndex(t.r0,t.g1,t.b0)]-n[this.getIndex(t.r0,t.g0,t.b0)];default:throw new Error("unexpected direction $direction")}}top(t,r,n,a){switch(r){case _.RED:return a[this.getIndex(n,t.g1,t.b1)]-a[this.getIndex(n,t.g1,t.b0)]-a[this.getIndex(n,t.g0,t.b1)]+a[this.getIndex(n,t.g0,t.b0)];case _.GREEN:return a[this.getIndex(t.r1,n,t.b1)]-a[this.getIndex(t.r1,n,t.b0)]-a[this.getIndex(t.r0,n,t.b1)]+a[this.getIndex(t.r0,n,t.b0)];case _.BLUE:return a[this.getIndex(t.r1,t.g1,n)]-a[this.getIndex(t.r1,t.g0,n)]-a[this.getIndex(t.r0,t.g1,n)]+a[this.getIndex(t.r0,t.g0,n)];default:throw new Error("unexpected direction $direction")}}getIndex(t,r,n){return(t<<lt*2)+(t<<lt+1)+t+(r<<lt)+r+n}}class ie{constructor(t=0,r=0,n=0,a=0,o=0,s=0,c=0){this.r0=t,this.r1=r,this.g0=n,this.g1=a,this.b0=o,this.b1=s,this.vol=c}}class ce{constructor(t,r){this.requestedCount=t,this.resultCount=r}}class le{constructor(t,r){this.cutLocation=t,this.maximum=r}}/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */class ue{static quantize(t,r){const a=new se().quantize(t,r);return ne.quantize(t,a,r)}}/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */class X{get primary(){return this.props.primary}get onPrimary(){return this.props.onPrimary}get primaryContainer(){return this.props.primaryContainer}get onPrimaryContainer(){return this.props.onPrimaryContainer}get secondary(){return this.props.secondary}get onSecondary(){return this.props.onSecondary}get secondaryContainer(){return this.props.secondaryContainer}get onSecondaryContainer(){return this.props.onSecondaryContainer}get tertiary(){return this.props.tertiary}get onTertiary(){return this.props.onTertiary}get tertiaryContainer(){return this.props.tertiaryContainer}get onTertiaryContainer(){return this.props.onTertiaryContainer}get error(){return this.props.error}get onError(){return this.props.onError}get errorContainer(){return this.props.errorContainer}get onErrorContainer(){return this.props.onErrorContainer}get background(){return this.props.background}get onBackground(){return this.props.onBackground}get surface(){return this.props.surface}get onSurface(){return this.props.onSurface}get surfaceVariant(){return this.props.surfaceVariant}get onSurfaceVariant(){return this.props.onSurfaceVariant}get outline(){return this.props.outline}get outlineVariant(){return this.props.outlineVariant}get shadow(){return this.props.shadow}get scrim(){return this.props.scrim}get inverseSurface(){return this.props.inverseSurface}get inverseOnSurface(){return this.props.inverseOnSurface}get inversePrimary(){return this.props.inversePrimary}static light(t){return X.lightFromCorePalette(S.of(t))}static dark(t){return X.darkFromCorePalette(S.of(t))}static lightContent(t){return X.lightFromCorePalette(S.contentOf(t))}static darkContent(t){return X.darkFromCorePalette(S.contentOf(t))}static lightFromCorePalette(t){return new X({primary:t.a1.tone(40),onPrimary:t.a1.tone(100),primaryContainer:t.a1.tone(90),onPrimaryContainer:t.a1.tone(10),secondary:t.a2.tone(40),onSecondary:t.a2.tone(100),secondaryContainer:t.a2.tone(90),onSecondaryContainer:t.a2.tone(10),tertiary:t.a3.tone(40),onTertiary:t.a3.tone(100),tertiaryContainer:t.a3.tone(90),onTertiaryContainer:t.a3.tone(10),error:t.error.tone(40),onError:t.error.tone(100),errorContainer:t.error.tone(90),onErrorContainer:t.error.tone(10),background:t.n1.tone(99),onBackground:t.n1.tone(10),surface:t.n1.tone(99),onSurface:t.n1.tone(10),surfaceVariant:t.n2.tone(90),onSurfaceVariant:t.n2.tone(30),outline:t.n2.tone(50),outlineVariant:t.n2.tone(80),shadow:t.n1.tone(0),scrim:t.n1.tone(0),inverseSurface:t.n1.tone(20),inverseOnSurface:t.n1.tone(95),inversePrimary:t.a1.tone(80)})}static darkFromCorePalette(t){return new X({primary:t.a1.tone(80),onPrimary:t.a1.tone(20),primaryContainer:t.a1.tone(30),onPrimaryContainer:t.a1.tone(90),secondary:t.a2.tone(80),onSecondary:t.a2.tone(20),secondaryContainer:t.a2.tone(30),onSecondaryContainer:t.a2.tone(90),tertiary:t.a3.tone(80),onTertiary:t.a3.tone(20),tertiaryContainer:t.a3.tone(30),onTertiaryContainer:t.a3.tone(90),error:t.error.tone(80),onError:t.error.tone(20),errorContainer:t.error.tone(30),onErrorContainer:t.error.tone(80),background:t.n1.tone(10),onBackground:t.n1.tone(90),surface:t.n1.tone(10),onSurface:t.n1.tone(90),surfaceVariant:t.n2.tone(30),onSurfaceVariant:t.n2.tone(80),outline:t.n2.tone(60),outlineVariant:t.n2.tone(30),shadow:t.n1.tone(0),scrim:t.n1.tone(0),inverseSurface:t.n1.tone(90),inverseOnSurface:t.n1.tone(20),inversePrimary:t.a1.tone(40)})}constructor(t){this.props=t}toJSON(){return{...this.props}}}/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */const he={desired:4,fallbackColorARGB:4282549748,filter:!0};function fe(e,t){return e.score>t.score?-1:e.score<t.score?1:0}class H{constructor(){}static score(t,r){const{desired:n,fallbackColorARGB:a,filter:o}={...he,...r},s=[],c=new Array(360).fill(0);let u=0;for(const[f,M]of t.entries()){const g=E.fromInt(f);s.push(g);const b=Math.floor(g.hue);c[b]+=M,u+=M}const h=new Array(360).fill(0);for(let f=0;f<360;f++){const M=c[f]/u;for(let g=f-14;g<f+16;g++){const b=Ot(g);h[b]+=M}}const l=new Array;for(const f of s){const M=Ot(Math.round(f.hue)),g=h[M];if(o&&(f.chroma<H.CUTOFF_CHROMA||g<=H.CUTOFF_EXCITED_PROPORTION))continue;const b=g*100*H.WEIGHT_PROPORTION,w=f.chroma<H.TARGET_CHROMA?H.WEIGHT_CHROMA_BELOW:H.WEIGHT_CHROMA_ABOVE,y=(f.chroma-H.TARGET_CHROMA)*w,k=b+y;l.push({hct:f,score:k})}l.sort(fe);const d=[];for(let f=90;f>=15;f--){d.length=0;for(const{hct:M}of l)if(d.find(b=>Nt(M.hue,b.hue)<f)||d.push(M),d.length>=n)break;if(d.length>=n)break}const p=[];d.length===0&&p.push(a);for(const f of d)p.push(f.toInt());return p}}H.TARGET_CHROMA=48;H.WEIGHT_PROPORTION=.7;H.WEIGHT_CHROMA_ABOVE=.3;H.WEIGHT_CHROMA_BELOW=.1;H.CUTOFF_CHROMA=5;H.CUTOFF_EXCITED_PROPORTION=.01;/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */function F(e){const t=dt(e),r=mt(e),n=gt(e),a=[t.toString(16),r.toString(16),n.toString(16)];for(const[o,s]of a.entries())s.length===1&&(a[o]="0"+s);return"#"+a.join("")}function de(e){e=e.replace("#","");const t=e.length===3,r=e.length===6,n=e.length===8;if(!t&&!r&&!n)throw new Error("unexpected hex "+e);let a=0,o=0,s=0;return t?(a=J(e.slice(0,1).repeat(2)),o=J(e.slice(1,2).repeat(2)),s=J(e.slice(2,3).repeat(2))):r?(a=J(e.slice(0,2)),o=J(e.slice(2,4)),s=J(e.slice(4,6))):n&&(a=J(e.slice(2,4)),o=J(e.slice(4,6)),s=J(e.slice(6,8))),(255<<24|(a&255)<<16|(o&255)<<8|s&255)>>>0}function J(e){return parseInt(e,16)}/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */async function me(e){const t=await new Promise((s,c)=>{const u=document.createElement("canvas"),h=u.getContext("2d");if(!h){c(new Error("Could not get canvas context"));return}const l=()=>{u.width=e.width,u.height=e.height,h.drawImage(e,0,0);let d=[0,0,e.width,e.height];const p=e.dataset.area;p&&/^\d+(\s*,\s*\d+){3}$/.test(p)&&(d=p.split(/\s*,\s*/).map(w=>parseInt(w,10)));const[f,M,g,b]=d;s(h.getImageData(f,M,g,b).data)};e.complete?l():e.onload=l}),r=[];for(let s=0;s<t.length;s+=4){const c=t[s],u=t[s+1],h=t[s+2];if(t[s+3]<255)continue;const d=ft(c,u,h);r.push(d)}const n=ue.quantize(r,128);return H.score(n)[0]}/**
 * @license
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */function Gt(e,t=[]){const r=S.of(e);return{source:e,schemes:{light:X.light(e),dark:X.dark(e)},palettes:{primary:r.a1,secondary:r.a2,tertiary:r.a3,neutral:r.n1,neutralVariant:r.n2,error:r.error},customColors:t.map(n=>ge(e,n))}}async function Lt(e,t=[]){const r=await me(e);return Gt(r,t)}function ge(e,t){let r=t.value;const n=r,a=e;t.blend&&(r=xt.harmonize(n,a));const s=S.of(r).a1;return{color:t,value:r,light:{color:s.tone(40),onColor:s.tone(100),colorContainer:s.tone(90),onColorContainer:s.tone(10)},dark:{color:s.tone(80),onColor:s.tone(20),colorContainer:s.tone(30),onColorContainer:s.tone(90)}}}function Pt(e){let t=JSON.parse(JSON.stringify(e.schemes));for(let r in t)for(let n in t[r])t[r][n]=F(t[r][n]);return t.dark.surfaceDim=F(e.palettes.neutral.tone(6)),t.dark.surface=F(e.palettes.neutral.tone(6)),t.dark.surfaceBright=F(e.palettes.neutral.tone(24)),t.dark.surfaceContainerLowest=F(e.palettes.neutral.tone(4)),t.dark.surfaceContainerLow=F(e.palettes.neutral.tone(10)),t.dark.surfaceContainer=F(e.palettes.neutral.tone(12)),t.dark.surfaceContainerHigh=F(e.palettes.neutral.tone(17)),t.dark.surfaceContainerHighest=F(e.palettes.neutral.tone(22)),t.dark.onSurface=F(e.palettes.neutral.tone(90)),t.dark.onSurfaceVariant=F(e.palettes.neutralVariant.tone(80)),t.dark.outline=F(e.palettes.neutralVariant.tone(60)),t.dark.outlineVariant=F(e.palettes.neutralVariant.tone(30)),t.light.surfaceDim=F(e.palettes.neutral.tone(87)),t.light.surface=F(e.palettes.neutral.tone(98)),t.light.surfaceBright=F(e.palettes.neutral.tone(98)),t.light.surfaceContainerLowest=F(e.palettes.neutral.tone(100)),t.light.surfaceContainerLow=F(e.palettes.neutral.tone(96)),t.light.surfaceContainer=F(e.palettes.neutral.tone(94)),t.light.surfaceContainerHigh=F(e.palettes.neutral.tone(92)),t.light.surfaceContainerHighest=F(e.palettes.neutral.tone(90)),t.light.onSurface=F(e.palettes.neutral.tone(10)),t.light.onSurfaceVariant=F(e.palettes.neutralVariant.tone(30)),t.light.outline=F(e.palettes.neutralVariant.tone(50)),t.light.outlineVariant=F(e.palettes.neutralVariant.tone(80)),t}async function pe(e){const t=e,r={light:{},dark:{}};try{if(typeof t=="string"&&/^\#[0-9a-f]+$/i.test(t)){let s=Gt(de(t));return Pt(s)}if(t.src){let s=await Lt(t);return Pt(s)}let n=new Blob;if(typeof t=="string"&&(n=await fetch(t).then(s=>s.blob())),t.size&&(n=t),t.files&&t.files[0]&&(n=t.files[0]),t.target&&t.target.files&&t.target.files[0]&&(n=t.target.files[0]),!n.size)return r;let a=new Image(64);a.src=URL.createObjectURL(n);let o=await Lt(a);return Pt(o)}catch{return r}}globalThis.materialDynamicColors=pe;

export default globalThis.materialDynamicColors;
function sleep(milliseconds) {
  const date = Date.now();
  let currentDate = null;
  do {
    currentDate = Date.now();
  } while (currentDate - date < milliseconds);
}

function f2() {
  for (var i = 0; i < 10; i++) {
    try {
      let z = arr[i].document.getElementById("add-to-ebooks-cart-button");
      let g = z.click();
      console.log("done " + i);
    } catch (err) {
      console.log("failed " + i);
    }
    sleep(500);
  }
}

function f1(j) {
  arr = [];
  for (var i = j; i < j + 10; i++) {
    z = window.open(x[i], "_blank");
    arr.push(z);
  }
}

function f3() {
  for (var i = 0; i < 10; i++) {
    arr[i].window.close();
  }
}

function f4() {
  for (let i = 0; i < 10; i++) {
    arr[i].window.location.reload();
  }
}

function process(ppp) {
  f1(ppp);
  alert(ppp);
  f2();
  alert("want to close?");
  f3();
}

function processBatch(a, b) {
  for (var i = a; i < b; i += 10) {
    process(i)
  }
}

function openRange(x, j, k) {
  for (var i = j; i < k; i++) {
    window.open(x[i], "_blank");
    if (i % 50 == 49) {
      alert("continue? " + i);
    }
  }
}

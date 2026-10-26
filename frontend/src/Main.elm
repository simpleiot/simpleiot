module Main exposing (Model, Msg, main)

import Browser
import Browser.Navigation as Nav exposing (Key)
import Effect
import Gen.Model
import Gen.Pages as Pages
import Gen.Route as Route
import Request
import Shared
import Url exposing (Url)
import View


main : Program Shared.Flags Model Msg
main =
    Browser.application
        { init = init
        , update = update
        , view = view
        , subscriptions = subscriptions
        , onUrlChange = ChangedUrl
        , onUrlRequest = ClickedLink
        }



-- INIT


type alias Model =
    { url : Url
    , key : Key
    , shared : Shared.Model
    , page : Pages.Model
    }


init : Shared.Flags -> Url -> Key -> ( Model, Cmd Msg )
init flags fullUrl key =
    let
        url =
            fromBase fullUrl

        ( shared, sharedCmd ) =
            Shared.init (Request.create () url key) flags

        ( page, effect ) =
            Pages.init (Route.fromUrl url) shared url key
    in
    ( Model url key shared page
    , Cmd.batch
        [ Cmd.map Shared sharedCmd
        , Effect.toCmd ( Shared, Page ) effect
        ]
    )



-- BASE PATH


{-| base is the path this UI is served under. Routes are matched without
it, and addresses are kept with it.
-}
base : String
base =
    "/old"


hasBase : Url -> Bool
hasBase url =
    url.path == base || String.startsWith (base ++ "/") url.path


fromBase : Url -> Url
fromBase url =
    if url.path == base then
        { url | path = "/" }

    else if String.startsWith (base ++ "/") url.path then
        { url | path = String.dropLeft (String.length base) url.path }

    else
        url


toBase : Url -> Url
toBase url =
    if hasBase url then
        url

    else
        { url | path = base ++ url.path }



-- UPDATE


type Msg
    = ChangedUrl Url
    | ClickedLink Browser.UrlRequest
    | Shared Shared.Msg
    | Page Pages.Msg


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        ClickedLink (Browser.Internal url) ->
            ( model
            , Nav.pushUrl model.key (Url.toString (toBase url))
            )

        ClickedLink (Browser.External url) ->
            ( model
            , Nav.load url
            )

        ChangedUrl fullUrl ->
            if hasBase fullUrl then
                changedUrl (fromBase fullUrl) model

            else
                -- elm-spa routes to paths without the base; keep the
                -- address under it so a reload returns to this UI
                ( model, Nav.replaceUrl model.key (Url.toString (toBase fullUrl)) )

        Shared sharedMsg ->
            let
                ( shared, sharedCmd ) =
                    Shared.update (Request.create () model.url model.key) sharedMsg model.shared

                ( page, effect ) =
                    Pages.init (Route.fromUrl model.url) shared model.url model.key
            in
            if page == Gen.Model.Redirecting_ then
                ( { model | shared = shared, page = page }
                , Cmd.batch
                    [ Cmd.map Shared sharedCmd
                    , Effect.toCmd ( Shared, Page ) effect
                    ]
                )

            else
                ( { model | shared = shared }
                , Cmd.map Shared sharedCmd
                )

        Page pageMsg ->
            let
                ( page, effect ) =
                    Pages.update pageMsg model.page model.shared model.url model.key
            in
            ( { model | page = page }
            , Effect.toCmd ( Shared, Page ) effect
            )


changedUrl : Url -> Model -> ( Model, Cmd Msg )
changedUrl url model =
    if url.path /= model.url.path then
        let
            ( page, effect ) =
                Pages.init (Route.fromUrl url) model.shared url model.key
        in
        ( { model | url = url, page = page }
        , Effect.toCmd ( Shared, Page ) effect
        )

    else
        ( { model | url = url }, Cmd.none )



-- VIEW


view : Model -> Browser.Document Msg
view model =
    Pages.view model.page model.shared model.url model.key
        |> View.map Page
        |> View.toBrowserDocument



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ Pages.subscriptions model.page model.shared model.url model.key |> Sub.map Page
        , Shared.subscriptions (Request.create () model.url model.key) model.shared |> Sub.map Shared
        ]
